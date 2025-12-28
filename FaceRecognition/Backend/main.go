package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	controller "github.com/kennethandrew67/go-backend/controller"
)

func main() {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173", "http://localhost:8080", "http://localhost:6379"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "Origin", "Accept"},
		AllowCredentials: true,
	}))

	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, Gin!",
		})
	})

	router.GET("/jobs", controller.GetTeachingJob)
	router.GET("/jobs/next", controller.GetNextTeachingRoom)

	router.Run(":8080")
}
