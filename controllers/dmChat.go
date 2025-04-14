package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shogoshima/divertidachat-backend/models"
	"github.com/shogoshima/divertidachat-backend/services"
	"gorm.io/gorm"
)

func CreateSingleChat(c *gin.Context) {
	// Extract the user ID from context and the target username
	currentUserID, _ := c.Get("id") // Ensure type conversion if necessary

	// Get the current user
	var currentUser models.User
	if err := services.DB.First(&currentUser, "id = ?", currentUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Parse request body
	type requestBody struct {
		Username string `json:"username"`
	}
	var body requestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	// Prevent chatting with oneself
	if currentUser.Username == body.Username {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot create a chat with yourself"})
		return
	}

	// Get the target user
	var targetUser models.User
	if err := services.DB.Where("username = ?", body.Username).First(&targetUser).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	type ChatIDResult struct {
		ChatID uuid.UUID `gorm:"column:chat_id"`
	}

	// Check if a chat between these two users already exists
	var result ChatIDResult
	err := services.DB.Raw(`
		SELECT cu1.chat_id FROM chat_users cu1 
		JOIN chat_users cu2 ON cu1.chat_id = cu2.chat_id
		JOIN chats ON chats.id = cu1.chat_id
		WHERE cu1.user_id = ? AND cu2.user_id = ? AND chats.is_group = false
	`, currentUser.ID, targetUser.ID).Scan(&result).Error

	fmt.Println("chatID after query: ", result.ChatID)

	if err == nil && result.ChatID != uuid.Nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Chat already exists"})
		return
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Create a new chat
	newChat := models.Chat{IsGroup: false}
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

	// Create Chat Details to return to user
	summary := models.ChatSummary{
		ChatID:    newChat.ID,
		IsGroup:   newChat.IsGroup,
		ChatName:  targetUser.Name,
		ChatPhoto: targetUser.PhotoURL,
	}

	c.JSON(http.StatusCreated, gin.H{"chat": summary})
}
