package main

import (
	"fmt"
	"log"
)

func main() {
	log.Printf("application starting on port %d", 8080)
	
	if err := connectDatabase("localhost", 5432); err != nil {
		log.Fatalf("startup failed: %v", err)
	}
	
	handleRequest("req-123", "user-456", 1500)
}

func handleRequest(requestID string, userID string, duration int) {
	log.Printf("processing request %s for user %s", requestID, userID)
	
	// Simulate processing
	if duration > 1000 {
		log.Printf("slow request: %s took %dms", requestID, duration)
	}
	
	log.Printf("request %s completed in %dms", requestID, duration)
}

func connectDatabase(host string, port int) error {
	log.Printf("connecting to database at %s:%d", host, port)
	
	// Simulate connection
	err := fmt.Errorf("connection refused")
	if err != nil {
		log.Printf("database connection failed: %v", err)
		return err
	}
	
	log.Println("database connection established")
	return nil
}

func processUser(userID string, age int) error {
	log.Printf("processing user %s with age %d", userID, age)
	
	// Business logic here
	if age < 18 {
		log.Printf("user %s is underage", userID)
		return fmt.Errorf("user too young")
	}
	
	log.Println("user processed successfully")
	return nil
}
