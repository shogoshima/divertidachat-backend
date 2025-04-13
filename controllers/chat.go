package controllers

import (
	"errors"
	"net/http"
	"strconv"

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
	chatSummary := models.ChatSummary{
		ChatID:    newChat.ID,
		IsGroup:   newChat.IsGroup,
		ChatName:  targetUser.Name,
		ChatPhoto: targetUser.PhotoURL,
	}

	c.JSON(http.StatusCreated, gin.H{"chat": chatSummary})
}

func GetChatSummaries(c *gin.Context) {
	// Extract the user ID from context
	userID, _ := c.Get("id")

	// Fetch chats that belong to this user
	var chats []models.Chat
	if err := services.DB.
		Select("chats.id, chats.name, chats.is_group").
		Joins("JOIN chat_users ON chats.id = chat_users.chat_id").
		Where("chat_users.user_id = ?", userID).
		Order("chats.updated_at DESC").
		Find(&chats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chats"})
		return
	}

	if len(chats) == 0 {
		c.JSON(http.StatusOK, gin.H{"chats": []models.ChatSummary{}})
		return
	}

	chatIDs := make([]uuid.UUID, len(chats))
	for i, chat := range chats {
		chatIDs[i] = chat.ID
	}

	// Fetch all participants grouped by chat
	type ChatParticipant struct {
		ChatID   uuid.UUID
		UserID   uuid.UUID
		Name     string
		Username string
		PhotoURL *string
	}

	var rawParticipants []ChatParticipant
	if err := services.DB.
		Table("chat_users").
		Select("chat_users.chat_id, users.id AS user_id, users.name, users.username, users.photo_url").
		Joins("JOIN users ON users.id = chat_users.user_id").
		Where("chat_users.chat_id IN ?", chatIDs).
		Find(&rawParticipants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch participants"})
		return
	}

	// Group by chatID
	participantsByChat := make(map[uuid.UUID][]models.UserPublicInfo)
	for _, p := range rawParticipants {
		participantsByChat[p.ChatID] = append(participantsByChat[p.ChatID], models.UserPublicInfo{
			ID:       p.UserID,
			Name:     p.Name,
			Username: p.Username,
			PhotoURL: p.PhotoURL,
		})
	}

	var chatSummaries []models.ChatSummary
	for _, chat := range chats {
		participants := participantsByChat[chat.ID]

		// If not a group, name = other participant's name
		var chatPhoto *string
		var chatName string = chat.Name
		if !chat.IsGroup {
			for _, p := range participants {
				if p.ID != userID {
					chatName = p.Name
					chatPhoto = p.PhotoURL
					break
				}
			}
		}

		// Fetch latest message for this chat
		var message models.Message
		if err := services.DB.
			Where("chat_id = ?", chat.ID).
			Order("sent_at DESC").
			Limit(1).
			First(&message).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
			return
		}

		var lastMsg *string
		if message.Text != "" {
			lastMsg = &message.Text
		}

		chatSummaries = append(chatSummaries, models.ChatSummary{
			ChatID:      chat.ID,
			ChatName:    chatName,
			IsGroup:     chat.IsGroup,
			LastMessage: lastMsg,
			ChatPhoto:   chatPhoto,
		})
	}

	c.JSON(http.StatusOK, gin.H{"chats": chatSummaries})
}

func GetSingleChatSummary(c *gin.Context) {
	// Extract the user ID from context
	userID, _ := c.Get("id")

	chatIDStr := c.Param("chatId")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	// Ensure the user belongs to this chat
	var chat models.Chat
	if err := services.DB.
		Select("id, name, is_group").
		Where("id = ? AND EXISTS (SELECT 1 FROM chat_users WHERE chat_id = ? AND user_id = ?)",
			chatID, chatID, userID).
		First(&chat).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found or access denied"})
		return
	}

	// Get the participants
	var participants []models.User
	if err := services.DB.
		Table("users").
		Select("users.id", "users.name", "users.username", "users.photo_url").
		Joins("JOIN chat_users ON chat_users.user_id = users.id").
		Where("chat_users.chat_id = ?", chatID).
		Find(&participants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch participants"})
		return
	}

	// If not a group, name = other participant's name
	var chatPhoto *string
	var chatName string = chat.Name
	if !chat.IsGroup {
		for _, p := range participants {
			if p.ID != userID {
				chatName = p.Name
				chatPhoto = p.PhotoURL
				break
			}
		}
	}

	// Fetch latest message for this chat
	var message models.Message
	if err := services.DB.
		Where("chat_id = ?", chatID).
		Order("sent_at DESC").
		Limit(1).
		First(&message).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	var lastMsg *string
	if message.Text != "" {
		lastMsg = &message.Text
	}

	chatSummary := models.ChatSummary{
		ChatID:      chatID,
		ChatName:    chatName,
		IsGroup:     chat.IsGroup,
		LastMessage: lastMsg,
		ChatPhoto:   chatPhoto,
	}

	c.JSON(http.StatusOK, gin.H{"chat": chatSummary})
}

func GetChatDetails(c *gin.Context) {
	// Extract and validate IDs
	userID, _ := c.Get("id")

	chatIDStr := c.Param("chatId")
	chatID, err := uuid.Parse(chatIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	// Ensure the user belongs to this chat
	var chat models.Chat
	if err := services.DB.
		Select("id, name, is_group").
		Where("id = ? AND EXISTS (SELECT 1 FROM chat_users WHERE chat_id = ? AND user_id = ?)",
			chatID, chatID, userID).
		First(&chat).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat not found or access denied"})
		return
	}

	// Fetch participants (single query)
	var participants []models.User
	if err := services.DB.
		Table("chat_users").
		Select("users.id, users.name, users.username, users.photo_url").
		Joins("JOIN users ON users.id = chat_users.user_id").
		Where("chat_users.chat_id = ?", chat.ID).
		Find(&participants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch participants"})
		return
	}

	var participantsPublicInfo []models.UserPublicInfo
	for _, p := range participants {
		participantsPublicInfo = append(participantsPublicInfo, models.UserPublicInfo{
			ID:       p.ID,
			Name:     p.Name,
			Username: p.Username,
			PhotoURL: p.PhotoURL,
			LastSeen: p.LastSeen,
		})
	}

	// Parse pagination params
	pageStr := c.DefaultQuery("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page parameter"})
		return
	}
	const pageSize = 40
	offset := (page - 1) * pageSize

	// Fetch paginated messages
	var messages []models.Message
	if err := services.DB.
		Select("id, chat_id, sender_id, text, sent_at").
		Where("chat_id = ?", chat.ID).
		Order("sent_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	// Return structured response
	c.JSON(http.StatusOK, gin.H{
		"chat": models.ChatDetails{
			ChatID:       chat.ID,
			Participants: participantsPublicInfo,
			Messages:     messages,
			Page:         page,
			PageSize:     pageSize,
		},
	})
}
