package controllers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	newChat := models.Chat{IsGroup: false, Name: targetUser.Name}
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
	chatDetails := models.ChatDetails{
		ChatID:       newChat.ID,
		ChatName:     newChat.Name,
		Messages:     []models.Message{},
		Participants: []models.User{currentUser, targetUser},
	}

	c.JSON(http.StatusCreated, gin.H{"chat": chatDetails})
}

func GetChatsAndTimestamps(c *gin.Context) {
	// Extract the user ID from context
	userID, _ := c.Get("id")

	// Get the chats for the user
	var chats []models.Chat
	if err := services.DB.
		Where("id IN (SELECT chat_id FROM chat_users WHERE user_id = ?)", userID).
		Find(&chats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chats"})
		return
	}

	var chatsWithTimestamps []models.ChatTimestamp
	for _, chat := range chats {
		chatTimestamp := models.ChatTimestamp{
			ChatID:    chat.ID,
			UpdatedAt: chat.UpdatedAt,
		}
		chatsWithTimestamps = append(chatsWithTimestamps, chatTimestamp)
	}

	c.JSON(http.StatusOK, gin.H{"timestamps": chatsWithTimestamps})
}

func GetAllUpdatedChats(c *gin.Context) {
	// Extract the user ID from context
	userID, _ := c.Get("id")

	// Get query parameters
	chatIDsStr := c.QueryArray("chat_ids")
	if len(chatIDsStr) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chat_ids query parameter is required"})
		return
	}

	var chatIDs []uuid.UUID
	for _, idStr := range chatIDsStr {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat_id: " + idStr})
			return
		}
		chatIDs = append(chatIDs, id)
	}

	sentAfterStr := c.Query("sent_after")
	if sentAfterStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sent_after query parameter is required"})
		return
	}

	sentAfter, err := time.Parse(time.RFC3339, sentAfterStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sent_after timestamp"})
		return
	}

	// Fetch chats that match the provided IDs, belong to this user, and have been updated after the provided timestamp
	var chats []models.Chat
	if err := services.DB.
		Where("id IN ?", chatIDs).
		Where("id IN (SELECT chat_id FROM chat_users WHERE user_id = ?)", userID).
		Where("updated_at > ?", sentAfter).
		Find(&chats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chats"})
		return
	}

	var chatsWithMessages []models.ChatDetails
	for _, chat := range chats {
		// Fetch the chat members using a subquery on chat_users
		var participants []models.User
		if err := services.DB.
			Where("id IN (?)",
				services.DB.Table("chat_users").
					Select("user_id").
					Where("chat_id = ?", chat.ID),
			).
			Find(&participants).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chat members"})
			return
		}

		// If it is not a group chat, make the name of the chat the name of the other user
		if !chat.IsGroup {
			for _, participant := range participants {
				if participant.ID != userID {
					chat.Name = participant.Name
				}
			}
		}

		// Get only the messages for the chat that were sent after the provided timestamp,
		// ordered by sent time descending
		var messages []models.Message
		if err := services.DB.
			Where("chat_id = ? AND sent_at > ?", chat.ID, sentAfter).
			Order("sent_at DESC").
			Find(&messages).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
			return
		}

		chatsWithMessages = append(chatsWithMessages, models.ChatDetails{
			ChatID:       chat.ID,
			ChatName:     chat.Name,
			Messages:     messages,
			Participants: participants,
		})
	}

	c.JSON(http.StatusOK, gin.H{"chats": chatsWithMessages})
}

func GetSingleUpdatedChat(c *gin.Context) {
	// Extract the user ID from context
	userID, _ := c.Get("id")

	// Get the chat ID from the URL parameter
	chatIDStr := c.Param("chatId")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	sentAfterStr := c.Query("sent_after")
	if sentAfterStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sent_after query parameter is required"})
		return
	}

	sentAfter, err := time.Parse(time.RFC3339, sentAfterStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sent_after timestamp"})
		return
	}

	var chat models.Chat
	if err := services.DB.
		Where("id = ? AND id IN (SELECT chat_id FROM chat_users WHERE user_id = ?)", chatID, userID).
		First(&chat).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found"})
		return
	}

	var participants []models.User
	if err := services.DB.
		Table("chat_users").
		Select("users.*").
		Joins("JOIN users ON users.id = chat_users.user_id").
		Where("chat_users.chat_id = ?", chat.ID).
		Find(&participants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch participants"})
		return
	}

	// If it is not a group chat, make the name of the chat the name of the other user
	if !chat.IsGroup {
		for _, participant := range participants {
			if participant.ID != userID {
				chat.Name = participant.Name
			}
		}
	}

	var messages []models.Message
	if err := services.DB.
		Where("chat_id = ? AND sent_at > ?", chat.ID, sentAfter).
		Order("sent_at DESC").
		Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chat": models.ChatDetails{
		ChatID:       chat.ID,
		ChatName:     chat.Name,
		Messages:     messages,
		Participants: participants,
	}})
}
