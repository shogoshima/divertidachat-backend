package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shogoshima/divertidachat-backend/services"
)

// Authenticate is a controller to authenticate a user.
// If the user does not exist, it creates a new user.
func Authenticate(c *gin.Context) {

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		c.Abort()
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
		c.Abort()
		return
	}

	// Verify the Google token and get user info
	googleToken := parts[1]
	user, userID, err := services.VerifyAndGetUserInfo(googleToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid google token"})
		c.Abort()
		return
	}

	// Create JWT token
	jwtToken, err := services.GenerateJWT(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT"})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": jwtToken, "user": user})

}
