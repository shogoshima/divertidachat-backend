package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/shogoshima/divertidachat-backend/models"
	"github.com/shogoshima/divertidachat-backend/services"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[string]*websocket.Conn) // Connected clients

var Broadcast = make(chan Message)              // Broadcast channel
var ActionBroadcast = make(chan Action)         // Action broadcast channel
var mutex = &sync.Mutex{}                       // Protect clients map
var PersistenceBroadcast = make(chan Message)   // Persistence channel
var NotificationsBroadcast = make(chan Message) // FCM Notifications channel

type Message struct {
	ID           uuid.UUID `json:"id"`
	Text         string    `json:"text"`
	SenderId     string    `json:"sender_id"`
	ChatId       uuid.UUID `json:"chat_id"`
	SentAt       time.Time `json:"sent_at"`
	TextFilterID int       `json:"text_filter_id"`
}

type Action struct {
	ChatId uuid.UUID `json:"chat_id"`
	Type   string    `json:"type"`
}

type Authorization struct {
	IdToken string `json:"id_token"`
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

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println("Failed to upgrade to WebSocket:", err)
		return
	}
	defer conn.Close()

	addClient(userID, conn)

	authenticate(userID, conn, c.Request.Context())

	for {
		var in Inbound
		err := conn.ReadJSON(&in)
		if err != nil {
			fmt.Println("Read error:", err)
			removeClient(userID)
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
				NotificationsBroadcast <- m
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
				NotificationsBroadcast <- msg
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

func HandleNotifications(ctx context.Context) {
	for {
		msg := <-NotificationsBroadcast

		chatID := msg.ChatId
		userID := msg.SenderId
		fmt.Println("Broadcasting notification to chat ID:", chatID)

		var tokens []string
		err := services.DB.
			Table("users").
			Joins("JOIN chat_users cu ON cu.user_id = users.id").
			Where("cu.chat_id = ?", chatID).
			Where("users.id <> ?", userID).
			Where("users.fcm_token IS NOT NULL").
			Pluck("users.fcm_token", &tokens).Error
		if err != nil {
			fmt.Println("Failed to find chat users:", err)
			continue
		}

		var sender models.User
		if err := services.DB.
			First(&sender, "id = ?", userID).
			Error; err != nil {
			fmt.Println("Failed to load sender user:", err)
		}

		for _, token := range tokens {
			// Construct the message.
			message := &messaging.Message{
				Notification: &messaging.Notification{
					Title: fmt.Sprintf("New Message from %s", sender.DisplayName),
					Body:  msg.Text,
				},
				Data: map[string]string{
					"type":        "message",
					"chat_id":     chatID.String(),
					"message_id":  msg.ID.String(),
					"sender_name": sender.DisplayName,
					"text":        msg.Text,
				},
				Token: token,
			}

			// Send the message.
			response, err := services.MessagingClient.Send(ctx, message)
			if err != nil {
				log.Fatalf("error sending message: %v\n", err)
				continue
			}
			log.Println("Successfully sent message:", response)
		}
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

func addClient(id string, conn *websocket.Conn) {
	mutex.Lock()
	clients[id] = conn
	fmt.Println("Client connected:", conn.RemoteAddr())
	mutex.Unlock()
}

func removeClient(id string) {
	mutex.Lock()
	delete(clients, id)
	fmt.Println("Client disconnected.")
	mutex.Unlock()
}

func authenticate(id string, conn *websocket.Conn, ctx context.Context) {
	var initialLoad Inbound
	err := conn.ReadJSON(&initialLoad)
	if err != nil {
		fmt.Println("Read error:", err)
		removeClient(id)
		return
	}

	if initialLoad.Type != "authentication" {
		sendError(conn, "invalid message payload")
		removeClient(id)
		return
	}

	var auth Authorization
	if err := json.Unmarshal(initialLoad.Data, &auth); err != nil {
		sendError(conn, "invalid message payload")
		removeClient(id)
		return
	}

	_, err = services.AuthClient.VerifyIDToken(ctx, auth.IdToken)
	if err != nil {
		sendError(conn, "invalid message payload")
		removeClient(id)
		return
	}

}
