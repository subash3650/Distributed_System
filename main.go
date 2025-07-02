package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Event struct {
	ID          string                 `json:"id" bson:"id"`
	UserID      string                 `json:"userId" bson:"userId"`
	Title       string                 `json:"title" bson:"title"`
	Description string                 `json:"description" bson:"description"`
	Completed   bool                   `json:"completed" bson:"completed"`
	CreatedAt   time.Time              `json:"createdAt" bson:"createdAt"`
	Topic       string                 `json:"topic,omitempty" bson:"topic,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty" bson:"data,omitempty"`
	Time        time.Time              `json:"time,omitempty" bson:"time,omitempty"`
}

var (
	Client          *mongo.Client
	EventCollection *mongo.Collection
	subscribers     = make(map[string][]chan Event)
	submutex        sync.RWMutex

	startTime      = time.Now()
	requestCounter int64
	inFlight       int64
)

func BenchmarkMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		atomic.AddInt64(&requestCounter, 1)
		log.Printf("[BACKEND %s] [BENCHMARK] %s %s took %v", os.Getenv("PORT"), c.Request.Method, c.Request.URL.Path, duration)
	}
}

func InFlightMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		atomic.AddInt64(&inFlight, 1)
		defer atomic.AddInt64(&inFlight, -1)
		c.Next()
	}
}

func loadHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"in_flight": atomic.LoadInt64(&inFlight),
	})
}

func benchmarkHandler(c *gin.Context) {
	uptime := time.Since(startTime)
	count := atomic.LoadInt64(&requestCounter)
	c.JSON(http.StatusOK, gin.H{
		"uptime":        uptime.String(),
		"request_count": count,
		"avg_req_per_s": float64(count) / uptime.Seconds(),
	})
}

func createEvent(c *gin.Context) {
	var newEvent Event
	if err := c.ShouldBindJSON(&newEvent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	newEvent.ID = uuid.New().String()
	newEvent.CreatedAt = time.Now()
	if _, err := EventCollection.InsertOne(context.TODO(), newEvent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}
	c.JSON(http.StatusCreated, newEvent)
}

func listEvents(c *gin.Context) {
	userID := c.Query("userId")
	filter := bson.M{}
	if userID != "" {
		filter["userId"] = userID
	}
	cursor, err := EventCollection.Find(context.TODO(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve events"})
		return
	}
	defer cursor.Close(context.TODO())

	var events []Event
	if err := cursor.All(context.TODO(), &events); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse events"})
		return
	}
	c.JSON(http.StatusOK, events)
}

func getEventByID(c *gin.Context) {
	id := c.Param("id")
	userID := c.Query("userId")
	filter := bson.M{"id": id}
	if userID != "" {
		filter["userId"] = userID
	}
	var event Event
	err := EventCollection.FindOne(context.TODO(), filter).Decode(&event)
	if err == mongo.ErrNoDocuments {
		c.JSON(http.StatusNotFound, gin.H{"message": "event not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to retrieve the event"})
		return
	}
	c.JSON(http.StatusOK, event)
}

func updateEvent(c *gin.Context) {
	id := c.Param("id")
	userID := c.Query("userId")
	var updated Event
	if err := c.ShouldBindJSON(&updated); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	filter := bson.M{"id": id}
	if userID != "" {
		filter["userId"] = userID
	}
	res, err := EventCollection.UpdateOne(
		context.TODO(),
		filter,
		bson.M{"$set": bson.M{
			"title":       updated.Title,
			"description": updated.Description,
			"completed":   updated.Completed,
		}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update the event"})
		return
	}
	if res.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "event not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "event updated successfully"})
}

func deleteEvent(c *gin.Context) {
	id := c.Param("id")
	userID := c.Query("userId")
	filter := bson.M{"id": id}
	if userID != "" {
		filter["userId"] = userID
	}
	res, err := EventCollection.DeleteOne(context.TODO(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete event"})
		return
	}
	if res.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "event not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func handlePublish(c *gin.Context) {
	var event Event
	if err := c.BindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if event.Topic == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Topic cannot be empty"})
		return
	}
	event.ID = uuid.New().String()
	event.Time = time.Now()

	if _, err := EventCollection.InsertOne(context.TODO(), event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save event"})
		return
	}

	submutex.RLock()
	count := 0
	for _, ch := range subscribers[event.Topic] {
		select {
		case ch <- event:
			count++
		default:
		}
	}
	submutex.RUnlock()
	fmt.Printf("Published and broadcasted event on topic '%s' to %d subscribers\n", event.Topic, count)
	c.JSON(http.StatusOK, gin.H{"status": "Event broadcasted"})
}

func handleGetEventsByTopic(c *gin.Context) {
	topic := c.Param("topic")
	cursor, err := EventCollection.Find(context.TODO(), bson.M{"topic": topic})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve events"})
		return
	}
	defer cursor.Close(context.TODO())

	var events []Event
	if err := cursor.All(context.TODO(), &events); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse events"})
		return
	}
	c.JSON(http.StatusOK, events)
}

func handleSubscribe(c *gin.Context) {
	topic := c.Param("topic")
	eventchan := make(chan Event, 10)

	submutex.Lock()
	subscribers[topic] = append(subscribers[topic], eventchan)
	fmt.Printf("[Subscribe] New subscriber for topic '%s'. Total subscribers: %d\n", topic, len(subscribers[topic]))
	submutex.Unlock()

	defer func() {
		submutex.Lock()
		chans := subscribers[topic]
		for i, ch := range chans {
			if ch == eventchan {
				subscribers[topic] = append(chans[:i], chans[i+1:]...)
				break
			}
		}
		submutex.Unlock()
		close(eventchan)
	}()

	timeout := time.After(60 * time.Second)

	c.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-eventchan:
			if ok {
				c.SSEvent("message", msg)
				return true
			}
			return false
		case <-timeout:
			fmt.Println("Subscription expired due to timeout.")
			return false
		}
	})
}

func memStatsHandler(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	c.JSON(http.StatusOK, gin.H{
		"alloc": m.Alloc,
	})
}

func BackendPortLogger(port string) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("[BACKEND %s] %s %s", port, c.Request.Method, c.Request.URL.Path)
		c.Next()
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		log.Fatal("MONGO_URI not set or empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	Client, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}

	db := Client.Database("fullApp")
	EventCollection = db.Collection("events")

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	r := gin.Default()
	r.Use(BackendPortLogger(port))
	r.Use(cors.Default())
	r.Use(BenchmarkMiddleware())
	r.Use(InFlightMiddleware())

	r.GET("/load", loadHandler)
	r.POST("/events", createEvent)
	r.GET("/events", listEvents)
	r.GET("/events/:id", getEventByID)
	r.PUT("/events/:id", updateEvent)
	r.DELETE("/events/:id", deleteEvent)

	r.POST("/publish", handlePublish)
	r.GET("/topic-events/:topic", handleGetEventsByTopic)
	r.GET("/subscribe/:topic", handleSubscribe)
	r.GET("/benchmark", benchmarkHandler)
	r.GET("/memstats", memStatsHandler)

	log.Printf("Backend server listening on :%s", port)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	serveErrors := make(chan error, 1)

	go func() {
		log.Printf("Server listening on :%s", port)
		serveErrors <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serveErrors:
		log.Fatalf("Server error: %v\n", err)
	case sig := <-stop:
		log.Printf("Received signal: %v - shutting down...", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown error: %v\n", err)
		} else {
			log.Println("Server shutdown complete")
		}
	}
}
