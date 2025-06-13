package models

import (
	"github.com/google/uuid"
)

// For communication with the frontend
type ChatDetails struct {
	ChatID       uuid.UUID       `json:"chat_id"`
	Messages     []Message       `json:"messages"`
	Participants []PublicProfile `json:"participants"`
	Page         int             `json:"page"`
	PageSize     int             `json:"page_size"`
}
