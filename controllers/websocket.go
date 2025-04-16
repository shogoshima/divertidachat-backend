package controllers

import (
	"encoding/json"
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

var Broadcast = make(chan Message)            // Broadcast channel
var ActionBroadcast = make(chan Action)       // Action broadcast channel
var mutex = &sync.Mutex{}                     // Protect clients map
var PersistenceBroadcast = make(chan Message) // Persistence channel

type Message struct {
	ID           uuid.UUID `json:"id"`
	Text         string    `json:"text"`
	SenderId     uuid.UUID `json:"sender_id"`
	ChatId       uuid.UUID `json:"chat_id"`
	SentAt       time.Time `json:"sent_at"`
	TextFilterID int       `json:"text_filter_id"`
}

type Action struct {
	ChatId uuid.UUID `json:"chat_id"`
	Type   string    `json:"type"`
}

type Inbound struct {
	Type string          `json:"type"` // e.g. "message", "action"
	Data json.RawMessage `json:"data"` // raw JSON payload
}

type WSResponse struct {
	Type    string `json:"type"`    // e.g., "message", "error", "action"
	Payload any    `json:"payload"` // actual content or error
}

func HandleWebSocket(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing user ID"})
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

	addClient(parsedUserID, conn)

	for {
		var in Inbound
		err := conn.ReadJSON(&in)
		if err != nil {
			fmt.Println("Read error:", err)
			removeClient(parsedUserID)
			break
		}

		switch in.Type {
		case "message":
			var m Message
			if err := json.Unmarshal(in.Data, &m); err != nil {
				sendError(conn, "invalid message payload")
				continue
			}

			// If no filtering, broadcast immediately:
			if m.TextFilterID == 0 {
				Broadcast <- m
				PersistenceBroadcast <- m
				continue
			}

			// Otherwise, do the GPT call asynchronously:
			go func(origConn *websocket.Conn, msg Message) {
				// call GPT
				var gptMessage []models.GPTMessage
				gptMessage = append(gptMessage, models.GPTMessage{
					Role:    "developer",
					Content: "You are an assistant that rewrites the user’s message in the same language as it was written, without adding quotation marks.",
				})
				gptMessage = append(gptMessage, models.GPTMessage{
					Role:    "user",
					Content: "Rewrite the following message " + TextFilters[m.TextFilterID].Command + ": '" + m.Text + "'",
				})

				resp, err := services.GetGPTResponse(gptMessage, msg.SenderId)
				if err != nil {
					// note: use a helper that locks and deletes if needed
					sendError(origConn, err.Error())
					return
				}

				// update the text and push into your pipelines
				msg.Text = resp
				Broadcast <- msg
				PersistenceBroadcast <- msg
			}(conn, m)

		case "action":
			var a Action
			if err := json.Unmarshal(in.Data, &a); err != nil {
				sendError(conn, "invalid action payload")
				continue
			}

			ActionBroadcast <- a

		default:
			sendError(conn, "unknown type "+in.Type)
		}

	}
}

func HandleActions() {
	for {
		action := <-ActionBroadcast

		chatID := action.ChatId
		fmt.Println("Broadcasting action to chat ID:", chatID)

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
			err := conn.WriteJSON(WSResponse{
				Type:    "action",
				Payload: action,
			})
			if err != nil {
				fmt.Println("Error writing message:", err)
				conn.Close()
				delete(clients, chatUser.UserID)
			}
		}
		mutex.Unlock()
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
			err := conn.WriteJSON(WSResponse{
				Type:    "message",
				Payload: msg,
			})
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
		msg := <-PersistenceBroadcast

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

		err = services.DB.Model(&models.Chat{}).
			Where("id = ?", msg.ChatId).
			UpdateColumn("updated_at", time.Now()).Error
		if err != nil {
			fmt.Println("Failed to update chat", err)
		}
		fmt.Println("Chat updated successfully")
	}
}

func sendError(conn *websocket.Conn, msg string) {
	conn.WriteJSON(WSResponse{
		Type:    "error",
		Payload: map[string]string{"message": msg},
	})
}

func addClient(id uuid.UUID, conn *websocket.Conn) {
	mutex.Lock()
	clients[id] = conn
	fmt.Println("Client connected:", conn.RemoteAddr())
	mutex.Unlock()
}

func removeClient(id uuid.UUID) {
	mutex.Lock()
	delete(clients, id)
	fmt.Println("Client disconnected.")
	mutex.Unlock()
}
