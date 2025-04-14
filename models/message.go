package models

import (
	"time"

	"github.com/google/uuid"
)

// Message represents an individual message within a chat.
// For the database
type Message struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ChatID   uuid.UUID `json:"chat_id" gorm:"type:uuid"`
	SenderID uuid.UUID `json:"sender_id" gorm:"type:uuid"`
	Text     string    `json:"text"`
	SentAt   time.Time `json:"sent_at" gorm:"autoCreateTime"`
	Seen     bool      `json:"seen" gorm:"default:false"`
}