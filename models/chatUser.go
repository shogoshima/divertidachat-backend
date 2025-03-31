package models

import (
	"time"

	"github.com/google/uuid"
)

// ChatUser is the join table that connects Users and Chats.
type ChatUser struct {
	ChatID   uuid.UUID `json:"chat_id" gorm:"type:uuid;primaryKey"`
	UserID   uuid.UUID `json:"user_id" gorm:"type:uuid;primaryKey"`
	JoinedAt time.Time `json:"joined_at" gorm:"autoCreateTime"`
}
