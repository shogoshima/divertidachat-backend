package models

import (
	"time"

	"github.com/google/uuid"
)

// Message represents an individual message within a chat.
type Message struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	ChatID   uuid.UUID `json:"chat_id" gorm:"type:uuid"`
	SenderID uuid.UUID `json:"sender_id" gorm:"type:uuid"`
	Text     string    `json:"text"`                          // SentAt uses autoCreateTime to capture when the message was sent.
	SentAt   time.Time `json:"sent_at" gorm:"autoCreateTime"` // Seen indicates if the message has been seen by its recipients.
	Seen     bool      `json:"seen" gorm:"default:false"`
}
