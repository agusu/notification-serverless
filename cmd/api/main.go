package main

import (
	"log"
	"os"

	"serverless-notification/cmd"
	"serverless-notification/cmd/api/routes"

	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func init() {
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			log.Println("Warning: .env file not found")
		}
	}
}

func main() {
	service := cmd.InitDependencies()

	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "notification-api",
		})
	})

	notificationRouteHandler := routes.NewNotificationRouteHandler(service)
	notificationRouteHandler.RegisterRoutes(router)

	if isLambda() {
		log.Println("Running in Lambda mode")
		ginLambda := ginadapter.New(router)
		lambda.Start(ginLambda.ProxyWithContext)
	} else {
		log.Println("Running locally on :8080")
		router.Run(":8080")
	}

}

func isLambda() bool {
	return os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != ""
}
