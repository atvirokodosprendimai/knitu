package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	log.Println("Starting Knit Agent...")

	// Main agent loop
	for {
		fmt.Println("Agent is running, doing agent things...")
		// In the future:
		// - Send heartbeat to server
		// - Listen for tasks from NATS
		// - Execute tasks (e.g., manage containers)
		time.Sleep(15 * time.Second)
	}
}
