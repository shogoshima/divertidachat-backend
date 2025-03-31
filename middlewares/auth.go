package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shogoshima/divertidachat-backend/services"
)

// AuthMiddleware verifies the Bearer token and fetches the user.
// It expects an "Authorization" header in the format "Bearer <token>".
func AuthMiddleware(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		c.Abort()
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
		c.Abort()
	}

	jwtToken := parts[1]

	userID, err := services.ValidateJWT(jwtToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		c.Abort()
	}

	// Set the user in the context so that handlers can access it.
	c.Set("id", userID)
	c.Next()
}
