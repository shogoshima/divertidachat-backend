package models

import (
	"github.com/google/uuid"
)

// For communication with the frontend
type ChatSummary struct {
	ChatID      uuid.UUID `json:"chat_id"`
	ChatName    string    `json:"chat_name"`
	IsGroup     bool      `json:"is_group"`
	ChatPhoto   string   `json:"chat_photo"`
	LastMessage *string   `json:"last_message"`
}
