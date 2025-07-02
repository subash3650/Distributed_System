package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

type Backend struct {
	URL        *url.URL
	MemoryUsed uint64
	Alive      bool
	mu         sync.RWMutex
}

var backendAddrs = []string{
	"http://localhost:5000",
	"http://localhost:5001",
	"http://localhost:5002",
}

var backends []*Backend

func updateLoad(b *Backend) {
	for {
		resp, err := http.Get(b.URL.String() + "/load")
		if err != nil {
			b.mu.Lock()
			b.Alive = false
			b.MemoryUsed = ^uint64(0)
			b.mu.Unlock()
			time.Sleep(2 * time.Second)
			continue
		}
		var stats struct {
			InFlight int64 `json:"in_flight"`
		}
		json.NewDecoder(resp.Body).Decode(&stats)
		resp.Body.Close()
		b.mu.Lock()
		b.MemoryUsed = uint64(stats.InFlight)
		b.Alive = true
		b.mu.Unlock()
		time.Sleep(200 * time.Millisecond)
	}
}

func getLeastMemoryBackend() *Backend {
	var min *Backend
	for _, b := range backends {
		b.mu.RLock()
		if b.Alive && (min == nil || b.MemoryUsed < min.MemoryUsed) {
			min = b
		}
		b.mu.RUnlock()
	}
	return min
}

func handler(w http.ResponseWriter, r *http.Request) {
	b := getLeastMemoryBackend()
	if b == nil {
		http.Error(w, "No backend available", http.StatusServiceUnavailable)
		return
	}
	log.Printf("[PROXY] %s %s -> backend %s (port %s)", r.Method, r.URL.Path, b.URL.String(), b.URL.Port())
	proxy := httputil.NewSingleHostReverseProxy(b.URL)
	proxy.ServeHTTP(w, r)
}

func main() {
	for _, addr := range backendAddrs {
		u, _ := url.Parse(addr)
		b := &Backend{URL: u, Alive: true}
		backends = append(backends, b)
		go updateLoad(b)
	}

	http.HandleFunc("/", handler)
	log.Println("Load balancer listening on :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
