package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"errors"

	"github.com/shogoshima/divertidachat-backend/models"
)

// https://developers.google.com/identity/sign-in/android/backend-auth?hl=pt-br
const googleTokenInfoURL = "https://oauth2.googleapis.com/tokeninfo?id_token=" // Google token info endpoint for verifying ID tokens:

// googleTokenInfo is the structure of the token info we get from Google.
type googleTokenInfo struct {
	Iss     string `json:"iss"`
	Sub     string `json:"sub"` // Unique identifier for the user
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// GetUserInfo verifies the provided token and returns the corresponding User record.
// If no user is found for the given token, it creates a new user based on the token info.
func VerifyAndGetUserInfo(token string) (models.User, error) {
	resp, err := http.Get(googleTokenInfoURL + token)
	if err != nil || resp.StatusCode != http.StatusOK {
		// If verification fails, return an empty user.
		return models.User{}, fmt.Errorf("failed to verify token: %w", err)
	}
	defer resp.Body.Close()

	var tokenInfo googleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return models.User{}, fmt.Errorf("failed to decode token info: %w", err)
	}

	// Check that the token was issued for your client ID
	if tokenInfo.Iss != "accounts.google.com" && tokenInfo.Iss != "https://accounts.google.com" {
		return models.User{}, fmt.Errorf("token issuer invalid: %s", tokenInfo.Iss)
	}

	var user models.User
	result := DB.Where("email = ?", tokenInfo.Email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// User not found, so create a new user.
			user = models.User{
				Name:     tokenInfo.Name,
				Email:    tokenInfo.Email,
				PhotoURL: tokenInfo.Picture,
				Username: generateUsername(tokenInfo.Email),
				LastSeen: time.Now(),
			}
			if err := DB.Create(&user).Error; err != nil {
				// In production, you might want to log the error
				return models.User{}, fmt.Errorf("failed to create user: %w", err)
			}
		} else {
			// For other DB errors, return an empty user.
			return models.User{}, fmt.Errorf("failed to query user: %w", result.Error)
		}
	}

	return user, nil

}

// generateUsername creates a simple username from the user's email.
// You can enhance this function to check for uniqueness, add random numbers, etc.
func generateUsername(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return "user"
}
