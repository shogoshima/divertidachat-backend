package models

import (
	"time"

	"github.com/google/uuid"
)

// ChatUser is the join table that connects Users and Chats.
// For the database
type ChatUser struct {
	ChatID   uuid.UUID `json:"chat_id" gorm:"type:uuid;primaryKey"`
	UserID   string    `json:"user_id" gorm:"primaryKey"`
	JoinedAt time.Time `json:"joined_at" gorm:"autoCreateTime"`

	Chat Chat `gorm:"constraint:OnDelete:CASCADE;"`
	User User `gorm:"constraint:OnDelete:CASCADE;"`
}
