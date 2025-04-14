package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user in the chat system.
// For the database
type User struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name       string     `json:"name"`
	Username   string     `json:"username" gorm:"uniqueIndex;not null"`
	Email      string     `json:"email" gorm:"uniqueIndex;not null"`
	PhotoURL   *string    `json:"photo_url"`
	LastSeen   *time.Time `json:"last_seen"`
	UsedTokens int        `json:"used_tokens" gorm:"default:0"`
	FCMToken   *string    `json:"fcm_token"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return
}
