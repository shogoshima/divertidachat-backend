package controllers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shogoshima/divertidachat-backend/models"
	"github.com/shogoshima/divertidachat-backend/services"
	"gorm.io/gorm"
)

func CreateGroupChat(c *gin.Context) {
	// Extract current user ID from context
	currentUserID, exists := c.Get("id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user context"})
		return
	}

	// Get the current user
	var currentUser models.User
	if err := services.DB.First(&currentUser, "id = ?", currentUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Parse request body
	type requestBody struct {
		Name      string   `json:"name" binding:"required"`
		Usernames []string `json:"usernames" binding:"required,min=1"`
	}
	var body requestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	// Ensure the creator isn't the only member
	// (we'll always add the creator ourselves below)
	if len(body.Usernames) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one other user is required"})
		return
	}

	// Load all target users by username
	var users []models.User
	if err := services.DB.Where("username IN ?", body.Usernames).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error loading users"})
		return
	}
	if len(users) != len(body.Usernames) {
		c.JSON(http.StatusNotFound, gin.H{"error": "One or more usernames not found"})
		return
	}

	// Create the group chat
	newChat := models.Chat{
		IsGroup:   true,
		Name:      body.Name,
		ChatPhoto: "https://picsum.photos/seed/" + body.Name + "/200",
	}
	if err := services.DB.Create(&newChat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat"})
		return
	}

	// Assemble ChatUser entries (including the creator)
	chatUsers := make([]models.ChatUser, 0, len(users)+1)
	chatUsers = append(chatUsers, models.ChatUser{
		ChatID: newChat.ID,
		UserID: currentUser.ID,
	})
	for _, u := range users {
		// Prevent duplicates if the creator username was included in the list
		if u.ID == currentUser.ID {
			continue
		}
		chatUsers = append(chatUsers, models.ChatUser{
			ChatID: newChat.ID,
			UserID: u.ID,
		})
	}

	// Bulk‐insert ChatUser records
	if err := services.DB.Create(&chatUsers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add users to group chat"})
		return
	}

	summary := models.ChatSummary{
		ChatID:    newChat.ID,
		IsGroup:   true,
		ChatName:  newChat.Name,
		ChatPhoto: newChat.ChatPhoto,
	}

	c.JSON(http.StatusCreated, gin.H{"chat": summary})
}

func AddUsersToGroupChat(c *gin.Context) {
	// Extract current user ID from context
	currentUserID, exists := c.Get("id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user context"})
		return
	}
	// Get the current user
	var currentUser models.User
	if err := services.DB.First(&currentUser, "id = ?", currentUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	chatIDParam := c.Param("chatId")
	chatID, err := uuid.Parse(chatIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	var chat models.Chat
	if err := services.DB.
		Joins("JOIN chat_users cu ON cu.chat_id = chats.id").
		Where("chats.id = ? AND cu.user_id = ? AND chats.is_group = true", chatID, currentUserID).
		First(&chat).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group chat not found or you’re not a member"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Parse request body
	type requestBody struct {
		Usernames []string `json:"usernames" binding:"required,min=1"`
	}
	var body requestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	var usersToAdd []models.User
	if err := services.DB.Where("username IN ?", body.Usernames).Find(&usersToAdd).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error loading users"})
		return
	}
	if len(usersToAdd) != len(body.Usernames) {
		c.JSON(http.StatusNotFound, gin.H{"error": "One or more usernames not found"})
		return
	}

	// Filter out users already in the chat
	var existing []uuid.UUID
	services.DB.
		Model(&models.ChatUser{}).
		Where("chat_id = ? AND user_id IN ?", chatID, getUserIDs(usersToAdd)).
		Pluck("user_id", &existing)

	existingSet := make(map[uuid.UUID]struct{}, len(existing))
	for _, id := range existing {
		existingSet[id] = struct{}{}
	}

	// Build ChatUser entries for only new users
	var newChatUsers []models.ChatUser
	for _, u := range usersToAdd {
		if _, found := existingSet[u.ID]; found {
			continue
		}
		newChatUsers = append(newChatUsers, models.ChatUser{
			ChatID: chatID,
			UserID: u.ID,
		})
	}
	if len(newChatUsers) == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "All users are already members"})
		return
	}

	// Bulk insert
	if err := services.DB.Create(&newChatUsers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add users to group chat"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Users added successfully"})
}

func LeaveGroupChat(c *gin.Context) {
	currentUserID, exists := c.Get("id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user context"})
		return
	}

	var currentUser models.User
	if err := services.DB.First(&currentUser, "id = ?", currentUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	chatIDParam := c.Param("chatId")
	chatID, err := uuid.Parse(chatIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	var chatUser models.ChatUser
	if err := services.DB.
		Joins("JOIN chats ON chats.id = chat_users.chat_id").
		Where("chat_users.user_id = ? AND chat_users.chat_id = ? AND chats.is_group = true", currentUser.ID, chatID).
		First(&chatUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group chat not found or you’re not a member"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if err := services.DB.Delete(&chatUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully removed user from group chat"})
}

func UpdateGroupChatInfo(c *gin.Context) {
	currentUserID, exists := c.Get("id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user context"})
		return
	}

	var currentUser models.User
	if err := services.DB.First(&currentUser, "id = ?", currentUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	chatIDParam := c.Param("chatId")
	chatID, err := uuid.Parse(chatIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat ID"})
		return
	}

	var chat models.Chat
	if err := services.DB.
		Joins("JOIN chat_users cu ON cu.chat_id = chats.id").
		Where("chats.id = ? AND cu.user_id = ? AND chats.is_group = true", chatID, currentUserID).
		First(&chat).
		Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Group chat not found or you’re not a member"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Parse request body
	type requestBody struct {
		Name string `json:"name" binding:"required"`
	}
	var body requestBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload: " + err.Error()})
		return
	}

	chat.Name = body.Name

	if err := services.DB.Save(&chat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Chat updated successfully"})

}
