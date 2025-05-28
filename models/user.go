package models

import (
	"time"
)

// Profile represents a user in the chat system.
// For the database
type User struct {
	ID          string     `json:"id" gorm:"primaryKey"`
	DisplayName string     `json:"display_name"`
	Username    string     `json:"username" gorm:"uniqueIndex;not null"`
	Email       string     `json:"email" gorm:"uniqueIndex;not null"`
	PhotoURL    string     `json:"photo_url"`
	LastSeen    *time.Time `json:"last_seen"`
	UsedTokens  int        `json:"used_tokens" gorm:"default:0"`
	FCMToken    *string    `json:"fcm_token"`
}
