✨ Distributed Event-Driven Task Management System in Go
This project is a scalable, event-driven backend platform built in Go, designed to demonstrate modern concurrency patterns, clean REST APIs, and horizontal scalability with a custom load balancer. 

🛠 Features
1. Task Management API
Create, update, retrieve, and delete tasks with persistent MongoDB storage.

2.Event Broadcasting System
Publish and subscribe to events in real time over Server-Sent Events (SSE).

3.Custom Load Balancer
Automatically distributes incoming requests to the backend instance with the lowest load.

4.Concurrency & Synchronization
Uses mutexes, channels, and context cancellation for safe concurrent processing.

5.Graceful Shutdown
Ensures all requests complete before stopping servers.

6.Automated Load Testing
Includes a Go-based test client to simulate high request volumes.

7.Scalable Design
Multiple backend instances run in parallel, demonstrating horizontal scaling.

🚀 Technologies
Go — Fast, efficient server logic
MongoDB — Flexible document storage for tasks and events
Gin — High-performance HTTP web framework
Server-Sent Events (SSE) — Real-time event streaming
Custom Load Balancer — Written in Go to distribute load dynamically

💡 Why This Project?
This project showcases:
How Go’s concurrency primitives (goroutines, channels, sync) make it easy to build highly concurrent, low-latency systems.
A fully working example of event-driven architecture with real-time updates.
Clean separation of concerns and maintainable modular design.
A starting point for building production-grade distributed services.

How to Run this project?
Open 5 terminaals 
1st : PORT=5000 go run main.go
2nd : PORT=5001 go run main.go
3rd : PORT=5002 go run main.go
4th navigate to the loadBalancer/ : go run balancer.go
5th navigate to the testData/ : go run Test.go

then if we see the log of balancer.go we can able to see which request is handled by which server.
