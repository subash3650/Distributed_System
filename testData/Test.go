package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const baseURL = "http://localhost:3000"

type Event struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"userId"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Completed   bool                   `json:"completed"`
	CreatedAt   time.Time              `json:"createdAt"`
	Topic       string                 `json:"topic,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Time        time.Time              `json:"time,omitempty"`
}

func main() {
	fmt.Println("Running Go test client...")

	var wg sync.WaitGroup
	numUsers := 10
	numIterations := 50

	for user := 1; user <= numUsers; user++ {
		userID := "user" + strconv.Itoa(user)
		wg.Add(1)
		go func(userID string) {
			defer wg.Done()
			userTest(userID, numIterations)
		}(userID)
	}

	wg.Wait()
	fmt.Println("All users finished.")

	resp, err := http.Get(baseURL + "/benchmark")
	if err != nil {
		fmt.Println("Error fetching benchmark:", err)
	} else {
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		fmt.Println("Benchmark results:", string(data))
	}
}

func userTest(userID string, iterations int) {
	var wg sync.WaitGroup
	for i := 0; i < iterations; i++ {
		wg.Add(5)
		eventTitle := fmt.Sprintf("Event %d by %s", i, userID)
		eventDesc := fmt.Sprintf("Description %d by %s", i, userID)

		go func() {
			defer wg.Done()
			eventID := createEvent(userID, eventTitle, eventDesc)
			if eventID == "" {
				return
			}

			go func() {
				defer wg.Done()
				updateEvent(userID, eventID)
			}()

			go func() {
				defer wg.Done()
				getEvent(userID, eventID)
			}()

			go func() {
				defer wg.Done()
				deleteEvent(userID, eventID)
			}()
		}()

		go func() {
			defer wg.Done()
			listEvents(userID)
		}()

		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
	}
	wg.Wait()
}

func createEvent(userID, title, description string) string {
	payload := map[string]interface{}{
		"userId":      userID,
		"title":       title,
		"description": description,
		"completed":   false,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(baseURL+"/events", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Println("Error creating event:", err)
		return ""
	}
	defer resp.Body.Close()

	var result Event
	data, _ := io.ReadAll(resp.Body)
	json.Unmarshal(data, &result)

	fmt.Printf("[%s] Created event: %s\n", userID, result.ID)
	return result.ID
}

func updateEvent(userID, id string) {
	payload := map[string]interface{}{
		"title":       "Updated event " + id,
		"description": "Updated description",
		"completed":   true,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest(http.MethodPut, baseURL+"/events/"+id+"?userId="+userID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("[%s] Error updating event: %v\n", userID, err)
		return
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	fmt.Printf("[%s] Updated event: %s\n", userID, string(data))
}

func getEvent(userID, id string) {
	resp, err := http.Get(baseURL + "/events/" + id + "?userId=" + userID)
	if err != nil {
		fmt.Printf("[%s] Error getting event: %v\n", userID, err)
		return
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	fmt.Printf("[%s] Fetched event: %s\n", userID, string(data))
}

func deleteEvent(userID, id string) {
	req, _ := http.NewRequest(http.MethodDelete, baseURL+"/events/"+id+"?userId="+userID, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("[%s] Error deleting event: %v\n", userID, err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("[%s] Deleted event: %s\n", userID, id)
}

func listEvents(userID string) {
	resp, err := http.Get(baseURL + "/events?userId=" + userID)
	if err != nil {
		fmt.Printf("[%s] Error listing events: %v\n", userID, err)
		return
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	fmt.Printf("[%s] All events: %s\n", userID, string(data))
}
