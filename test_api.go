package main

import (
	"fmt"
	"log"
	"os"

	"github.com/getzep/zep-web-interface/internal/zepapi"
)

func main() {
	// Test API connectivity
	apiURL := os.Getenv("ZEP_API_URL")
	apiKey := os.Getenv("ZEP_API_KEY")
	
	if apiURL == "" || apiKey == "" {
		log.Fatal("Please set ZEP_API_URL and ZEP_API_KEY environment variables")
	}
	
	fmt.Printf("Testing connection to Zep API at: %s\n", apiURL)
	
	client := zepapi.NewClient(apiURL, apiKey, "")
	
	// Test health endpoint
	fmt.Println("Testing health endpoint...")
	health, err := client.Health()
	if err != nil {
		log.Fatalf("Health check failed: %v", err)
	}
	fmt.Printf("Health check passed: %+v\n", health)
	
	// Test sessions endpoint
	fmt.Println("Testing sessions endpoint...")
	sessions, err := client.GetSessions()
	if err != nil {
		log.Fatalf("Get sessions failed: %v", err)
	}
	fmt.Printf("Found %d sessions\n", len(sessions))
	
	// Test users endpoint
	fmt.Println("Testing users endpoint...")
	users, err := client.GetUsers()
	if err != nil {
		log.Fatalf("Get users failed: %v", err)
	}
	fmt.Printf("Found %d users\n", len(users))
	
	fmt.Println("All API tests passed!")
}