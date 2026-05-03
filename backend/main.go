package main

import (
	"cancer-to-hell/handlers"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[main] No .env file found — using environment variables")
	}

	r := gin.Default()

	origins := []string{"http://localhost:3000"}
	if env := os.Getenv("FRONTEND_ORIGINS"); env != "" {
		origins = strings.Split(env, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
	}

	allowSuffix := strings.TrimSpace(os.Getenv("FRONTEND_ORIGIN_SUFFIX")) // e.g. ".vercel.app"

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			if slices.Contains(origins, origin) {
				return true
			}
			if allowSuffix != "" && strings.HasSuffix(origin, allowSuffix) {
				return true
			}
			return false
		},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "Cancer to Hell is running"})
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
