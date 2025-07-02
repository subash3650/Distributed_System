# ✨ Distributed Event-Driven Task Management System in Go

This project is a scalable, event-driven backend platform built in Go, designed to demonstrate modern concurrency patterns, clean REST APIs, and horizontal scalability with a custom load balancer.

---

## 🛠 Features

- **Task Management API:**  
  Create, update, retrieve, and delete tasks with persistent MongoDB storage.

- **Event Broadcasting System:**  
  Publish and subscribe to events in real time over Server-Sent Events (SSE).

- **Custom Load Balancer:**  
  Automatically distributes incoming requests to the backend instance with the lowest load.

- **Concurrency & Synchronization:**  
  Uses mutexes, channels, and context cancellation for safe concurrent processing.

- **Graceful Shutdown:**  
  Ensures all requests complete before stopping servers.

- **Automated Load Testing:**  
  Includes a Go-based test client to simulate high request volumes.

- **Scalable Design:**  
  Multiple backend instances run in parallel, demonstrating horizontal scaling.

---

## 🚀 Technologies

- **Go:** Fast, efficient server logic
- **MongoDB:** Flexible document storage for tasks and events
- **Gin:** High-performance HTTP web framework
- **Server-Sent Events (SSE):** Real-time event streaming
- **Custom Load Balancer:** Written in Go to distribute load dynamically

---

## 💡 Why This Project?

This project showcases:

- How Go’s concurrency primitives (goroutines, channels, sync) make it easy to build highly concurrent, low-latency systems.
- A fully working example of event-driven architecture with real-time updates.
- Clean separation of concerns and maintainable modular design.
- A starting point for building production-grade distributed services.

---

## 🚦 How to Run This Project

Open **five terminals** and run the following commands:

```sh
# Terminal 1
PORT=5000 go run main.go

# Terminal 2
PORT=5001 go run main.go

# Terminal 3
PORT=5002 go run main.go

# Terminal 4 (navigate to loadBalancer/)
go run balancer.go

# Terminal 5 (navigate to testData/)
go run Test.go
```

---

## 📊 Observing the Load Balancer

Check the logs in the terminal running `balancer.go` to see which backend server handled each request.  
This demonstrates dynamic load distribution based on real-time server load.

---

## 📁 Project Structure

```
.
├── main.go           # Backend server code
├── loadBalancer/
│   └── balancer.go   # Load balancer code
├── testData/
│   └── Test.go       # Load testing client
├── README.md
└── ...
```

---

## 📝 License

MIT License. See [LICENSE](LICENSE) for details.
