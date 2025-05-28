package controllers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/shogoshima/divertidachat-backend/models"
	"github.com/shogoshima/divertidachat-backend/services"
)

// helper to extract IDs
func getUserIDs(users []models.User) []string {
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.ID
	}
	return ids
}

func GetUserByUsername(c *gin.Context) {
	username := c.Param("username")
	var user models.User
	if err := services.DB.
		Where("username = ?", username).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": models.PublicProfile{
		ID:          user.ID,
		DisplayName: user.DisplayName,
		Username:    user.Username,
		PhotoURL:    user.PhotoURL,
		LastSeen:    user.LastSeen,
	}})
}

// UpdateUser updates a user's profile.
// The user must be authenticated; additional checks can be done to ensure users only update their own profile.
func UpdateUser(c *gin.Context) {
	user, _ := c.Get("currentUser")
	CurrentUser := user.(models.User)

	var input models.PublicProfile
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update allowed fields
	CurrentUser.Username = input.Username

	var userExists models.User
	err := services.DB.Where("username = ?", CurrentUser.Username).First(&userExists).Error
	if err == nil {
		// Found an existing user — this is a conflict
		c.JSON(http.StatusConflict, gin.H{"error": "User with this username already exists"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Any other DB error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if err := services.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": models.PublicProfile{
		ID:          CurrentUser.ID,
		DisplayName: CurrentUser.DisplayName,
		Username:    CurrentUser.Username,
		PhotoURL:    CurrentUser.PhotoURL,
		LastSeen:    CurrentUser.LastSeen,
	}})
}

func DeleteUser(c *gin.Context) {
	user, _ := c.Get("currentUser")
	CurrentUser := user.(models.User)

	// Gather all one‑on‑one chat IDs the user is in
	var dmChatIDs []uuid.UUID
	if err := services.DB.
		Model(&models.ChatUser{}).
		Joins("JOIN chats ON chats.id = chat_users.chat_id").
		Where("chat_users.user_id = ? AND chats.is_group = false", CurrentUser.ID).
		Pluck("chat_users.chat_id", &dmChatIDs).
		Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query user chats"})
		return
	}

	// Delete the user (this cascades to ChatUser via OnDelete:CASCADE on UserID)
	if err := services.DB.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	// Delete any DMs that the user was part of
	// (only non‑group chats, per your requirement)
	if len(dmChatIDs) > 0 {
		if err := services.DB.
			Where("id IN ?", dmChatIDs).
			Delete(&models.Chat{}).
			Error; err != nil {
			// Non‑fatal: user is gone, but log the error
			log.Printf("failed to delete DMs for deleted user %s: %v", CurrentUser.ID, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "User and their one-to-one chats deleted successfully"})
}

func GetAuthenticatedUser(c *gin.Context) {
	user, _ := c.Get("currentUser")
	CurrentUser := user.(models.User)

	now := time.Now()
	CurrentUser.LastSeen = &now
	if err := services.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": models.PublicProfile{
		ID:          CurrentUser.ID,
		DisplayName: CurrentUser.DisplayName,
		Username:    CurrentUser.Username,
		PhotoURL:    CurrentUser.PhotoURL,
		LastSeen:    CurrentUser.LastSeen,
	}})
}

func ResetGPTUsage() {
	// Actually perform the update
	err := services.DB.Model(&models.User{}).Where("1 = 1").UpdateColumn("used_tokens", 0).Error
	if err != nil {
		fmt.Println("Failed to reset used_tokens:", err)
		return
	}

	fmt.Println("Successfully reset used_tokens for all users")
}