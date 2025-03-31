package models

import "github.com/google/uuid"

type ChatMessages struct {
	ChatID   uuid.UUID `json:"chat_id" gorm:"type:uuid"`
	ChatName string    `json:"chat_name"`
	Messages []Message `json:"messages"`
}
