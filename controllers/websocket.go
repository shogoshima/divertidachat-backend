package controllers

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/shogoshima/divertidachat-backend/models"
	"github.com/shogoshima/divertidachat-backend/services"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[uuid.UUID]*websocket.Conn) // Connected clients
var Broadcast = make(chan Message)                // Broadcast channel
var mutex = &sync.Mutex{}                         // Protect clients map
var persistenceBroadcast = make(chan Message)     // Persistence channel
var persistenceMutex = &sync.Mutex{}              // Protect persistence operations

type Message struct {
	ID       uuid.UUID `json:"id"`
	Text     string    `json:"text"`
	SenderId uuid.UUID `json:"sender_id"`
	ChatId   uuid.UUID `json:"chat_id"`
	ChatName string    `json:"chat_name"`
	SentAt   time.Time `json:"sent_at"`
}

func HandleWebSocket(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing chat ID"})
		return
	}

	// Parse the user ID from the URL parameter
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println("Failed to upgrade to WebSocket:", err)
		return
	}
	defer conn.Close()

	mutex.Lock()
	clients[parsedUserID] = conn
	fmt.Println("Client connected:", conn.RemoteAddr())
	mutex.Unlock()

	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			mutex.Lock()
			delete(clients, parsedUserID)
			mutex.Unlock()
			fmt.Println("Client disconnected:", err)
			fmt.Println("Read error:", err)
			break
		}

		var gptMessage []models.GPTMessage
		gptMessage = append(gptMessage, models.GPTMessage{
			Role:    "user",
			Content: `Reescreva a seguinte mensagem com um estereótipo de "diva": ` + msg.Text,
		})
		gptResponse, err := services.GetGPTResponse(gptMessage)
		if err != nil {
			fmt.Println("Error getting GPT response:", err)
			continue
		}

		msg.Text = gptResponse

		fmt.Println("Received message:", msg.Text)
		Broadcast <- msg
		persistenceBroadcast <- msg
	}
}

func HandleMessages() {
	for {
		msg := <-Broadcast

		chatID := msg.ChatId
		fmt.Println("Broadcasting message to chat ID:", chatID)

		// Find all users in the chat (outside of mutex lock)
		var chatUsers []models.ChatUser
		err := services.DB.Where("chat_id = ?", chatID).Find(&chatUsers).Error
		if err != nil {
			fmt.Println("Failed to find chat users:", err)
			continue
		}

		// Lock the clients map before iterating
		mutex.Lock()
		for _, chatUser := range chatUsers {
			conn, ok := clients[chatUser.UserID]
			if !ok {
				fmt.Println("Client not found for user ID:", chatUser.UserID)
				continue
			}

			// Send message
			fmt.Printf("Sending message to user ID %d: %+v\n", chatUser.UserID, msg)
			err := conn.WriteJSON(msg)
			if err != nil {
				fmt.Println("Error writing message:", err)
				conn.Close()
				delete(clients, chatUser.UserID)
			}
		}
		mutex.Unlock()
	}
}

func HandlePersistence() {
	// This function is responsible for persisting messages to the database.
	// You can implement this function to save messages to your database.
	// For example, you can use GORM to save the message to a "messages" table.
	for {
		msg := <-persistenceBroadcast

		// Lock the persistence operations to prevent concurrent writes
		persistenceMutex.Lock()

		// Create a new message record in the database
		message := models.Message{
			ID:       msg.ID,
			Text:     msg.Text,
			SenderID: msg.SenderId,
			ChatID:   msg.ChatId,
			SentAt:   time.Now(),
		}

		err := services.DB.Create(&message).Error
		if err != nil {
			fmt.Println("Failed to save message to database:", err)
		}
		fmt.Println("Message saved to database:", message)

		// Unlock the persistence operations
		persistenceMutex.Unlock()
	}
}
