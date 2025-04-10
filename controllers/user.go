package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/shogoshima/divertidachat-backend/models"
	"github.com/shogoshima/divertidachat-backend/services"
)

// GetUser retrieves a user by ID.
// Only authenticated users can view user details.
func GetUserByID(c *gin.Context) {
	id := c.Param("userId")
	var user models.User
	if err := services.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func GetUserByUsername(c *gin.Context) {
	username := c.Param("username")
	var user models.User
	if err := services.DB.Where("username = ?", username).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

// UpdateUser updates a user's profile.
// The user must be authenticated; additional checks can be done to ensure users only update their own profile.
func UpdateUser(c *gin.Context) {
	id, _ := c.Get("id")
	var user models.User
	if err := services.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update allowed fields.
	user.Name = input.Name
	user.Username = input.Username
	user.PhotoURL = input.PhotoURL
	// Email is generally not updated if it's used as a unique identifier.
	if err := services.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func GetAuthenticatedUser(c *gin.Context) {
	id, _ := c.Get("id")
	var user models.User
	if err := services.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
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

func UpdateFCMToken(c *gin.Context) {
	id, _ := c.Get("id")
	var fcmToken models.FCMToken

	if err := c.BindJSON(&fcmToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "FCM Token not provided"})
	}

	err := services.DB.Model(&models.User{}).
		Where("id = ?", id).
		UpdateColumn("fcm_token", fcmToken.Token).Error
	if err != nil {
		fmt.Println("Failed to update fcm token", err)
	}
	fmt.Println("Token updated successfully")

	c.JSON(http.StatusOK, gin.H{"message": "Token updated successfully"})
}
