package models

import (
	"time"

	"github.com/google/uuid"
)

// Chat represents a conversation which could be either one-on-one or a group chat.
// For the database
type Chat struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string    `json:"name,omitempty"`
	IsGroup   bool      `json:"is_group" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	ChatPhoto string    `json:"chat_photo"`

	Messages  []Message  `gorm:"constraint:OnDelete:CASCADE;"`
	ChatUsers []ChatUser `gorm:"constraint:OnDelete:CASCADE;"`
}