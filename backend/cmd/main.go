package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("Starting AIPower-Efficiency-Pilot Backend...")

	// Initialize Gin router
	r := gin.Default()

	// Setup basic health check route
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "UP",
			"component": "aipower-efficiency-pilot-backend",
		})
	})

	// TODO: Initialize Kubernetes client
	// TODO: Initialize Redis client
	// TODO: Initialize MySQL client

	// Start server in a goroutine
	go func() {
		if err := r.Run(":8080"); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down AIPower-Efficiency-Pilot Backend...")
}
