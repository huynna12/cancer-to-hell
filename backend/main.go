package main

import (
	"cancer-to-hell/db"
	"cancer-to-hell/handlers"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[main] No .env file found — using environment variables")
	}

	db.Init("sessions.jsonl")

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "Cancer to Hell is running"})
	})

	r.GET("/api/v1/sessions", func(c *gin.Context) {
		c.JSON(200, db.List())
	})

	r.POST("/api/v1/decision-card", handlers.StartDebate)
	r.POST("/api/v1/decision-card/stream", handlers.StreamDebate)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("[main] Server running on port %s", port)
	r.Run(":" + port)
}
