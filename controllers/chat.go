package controllers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shogoshima/divertidachat-backend/models"
	"github.com/shogoshima/divertidachat-backend/services"
	"gorm.io/gorm"
)

func CreateSingleChat(c *gin.Context) {
	// Extract the user ID from context and the target username
	currentUserID, _ := c.Get("id") // Ensure type conversion if necessary
	username := c.Param("username")

	// Get the current user
	var currentUser models.User
	if err := services.DB.First(&currentUser, "id = ?", currentUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Prevent chatting with oneself
	if currentUser.Username == username {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot create a chat with yourself"})
		return
	}

	// Get the target user
	var targetUser models.User
	if err := services.DB.Where("username = ?", username).First(&targetUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check if a chat between these two users already exists
	var chatID uint
	err := services.DB.Raw(`
    SELECT cu1.chat_id FROM chat_users cu1 
    JOIN chat_users cu2 ON cu1.chat_id = cu2.chat_id
    JOIN chats ON chats.id = cu1.chat_id
    WHERE cu1.user_id = ? AND cu2.user_id = ? AND chats.is_group = false
`, currentUser.ID, targetUser.ID).Scan(&chatID).Error

	if err == nil && chatID != 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Chat already exists"})
		return
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Create a new chat
	newChat := models.Chat{IsGroup: false, Name: strings.Split(currentUser.Name, " ")[0] + " & " + strings.Split(targetUser.Name, " ")[0]}
	if err := services.DB.Create(&newChat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat"})
		return
	}

	// Add users to the chat in the ChatUser table
	chatUsers := []models.ChatUser{
		{ChatID: newChat.ID, UserID: currentUser.ID},
		{ChatID: newChat.ID, UserID: targetUser.ID},
	}
	if err := services.DB.Create(&chatUsers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add users to chat"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"chat": newChat})
}

func GetChats(c *gin.Context) {
	// Extract the user ID from context
	userID, _ := c.Get("id")

	// Get the chats for the user
	var chats []models.Chat
	if err := services.DB.Where("id IN (SELECT chat_id FROM chat_users WHERE user_id = ?)", userID).Find(&chats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chats"})
		return
	}

	var chatsWithMessages []models.ChatMessages = make([]models.ChatMessages, 0, len(chats))

	// Populate the chats with messages
	for i := range chats {
		var messages []models.Message
		if err := services.DB.Where("chat_id = ?", chats[i].ID).Order("sent_at DESC").Find(&messages).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
			return
		}
		chatsWithMessages = append(chatsWithMessages, models.ChatMessages{
			ChatID:   chats[i].ID,
			ChatName: chats[i].Name,
			Messages: messages,
		})
	}

	c.JSON(http.StatusOK, gin.H{"chats": chatsWithMessages})
}
