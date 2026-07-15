package main

import (
	"log"
	"prime-customer-care/config"
	"prime-customer-care/internal/middleware"
	"prime-customer-care/internal/routes"
	"prime-customer-care/internal/services/cronjob"
	_ "prime-customer-care/internal/services/cronjob-service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	config.Initialize()
	ginEngine := gin.Default()

	middleware.RegisterMiddlewares(ginEngine)

	routes.RegisterRoutes(ginEngine)
	cronjob.AutoStartCronJobs()
	port := "9117"
	log.Printf("Starting server on port %s\n", port)
	if err := ginEngine.Run(":" + port); err != nil {
		log.Fatalf("Could not start server: %s\n", err)
	}
}
