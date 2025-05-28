package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shogoshima/divertidachat-backend/models"
	"github.com/shogoshima/divertidachat-backend/services"
)

func UpdateFCMToken(c *gin.Context) {
	user, _ := c.Get("currentUser")
	CurrentUser := user.(models.User)

	var fcmToken models.FCMToken

	if err := c.BindJSON(&fcmToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "FCM Token not provided"})
	}

	err := services.DB.Model(&models.User{}).
		Where("id = ?", CurrentUser.ID).
		UpdateColumn("fcm_token", fcmToken.Token).Error
	if err != nil {
		fmt.Println("Failed to update fcm token", err)
	}
	fmt.Println("Token updated successfully")

	c.JSON(http.StatusOK, gin.H{"message": "Token updated successfully"})
}

func DeleteFCMToken(c *gin.Context) {
	user, _ := c.Get("currentUser")
	CurrentUser := user.(models.User)

	err := services.DB.Model(&models.User{}).
		Where("id = ?", CurrentUser.ID).
		UpdateColumn("fcm_token", nil).Error
	if err != nil {
		fmt.Println("Failed to delete fcm token", err)
	}
	fmt.Println("Token updated successfully")

	c.JSON(http.StatusOK, gin.H{"message": "Token updated successfully"})
}
