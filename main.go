package main

import (
	"log"

	"github.com/robfig/cron/v3"
	"github.com/shogoshima/divertidachat-backend/controllers"
	"github.com/shogoshima/divertidachat-backend/middlewares"
	"github.com/shogoshima/divertidachat-backend/services"

	"github.com/gin-gonic/gin"
)

func init() {
	services.LoadEnvs()
}

func main() {
	routes := gin.Default()

	// Initialize the database connection
	if err := services.InitializeDB(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Initialize cron job to reset all user usage
	c := cron.New()
	c.AddFunc("@midnight", controllers.ResetGPTUsage)
	c.Start()

	// Start goroutines for handling WebSocket messages and persistence
	go controllers.HandleMessages()
	go controllers.HandleActions()
	go controllers.HandlePersistence()

	// WebSocket connection for real-time chat
	routes.GET("/ws/:userId", controllers.HandleWebSocket)

	// Authentication routes
	routes.POST("/auth/login", controllers.Authenticate)

	// User routes
	userRoutes := routes.Group("/users")
	userRoutes.Use(middlewares.AuthMiddleware)
	{
		userRoutes.GET("/:username", controllers.GetUserByUsername) // Get user by username
		userRoutes.GET("/me", controllers.GetAuthenticatedUser)     // Get authenticated user
		userRoutes.PUT("/me", controllers.UpdateUser)               // Update authenticated user
		userRoutes.DELETE("/me", controllers.DeleteUser)            // Delete authenticated user
	}

	// Chat routes
	chatRoutes := routes.Group("/chats")
	chatRoutes.Use(middlewares.AuthMiddleware)
	{
		chatRoutes.GET("/textfilters", controllers.GetTextFilters) // Get text filters

		chatRoutes.GET("/summaries", controllers.GetChatSummaries) // Get all updated chats
		chatRoutes.GET("/summaries/:chatId", controllers.GetSingleChatSummary)
		chatRoutes.GET("/:chatId", controllers.GetChatDetails) // Get messages from a specific chat

		chatRoutes.POST("/dm", controllers.CreateSingleChat) // Create a new chat

		chatRoutes.POST("/group", controllers.CreateGroupChat)             // Create a new group chat
		chatRoutes.POST("/group/:chatId", controllers.AddUsersToGroupChat) // Add new users to group chat
		chatRoutes.PUT("/group/:chatId", controllers.UpdateGroupChatInfo)  // Update group chat info (name, photo?)
		chatRoutes.PUT("/group/leave/:chatId", controllers.LeaveGroupChat) // Leave from group chat
	}

	routes.Run(":8080")
}
