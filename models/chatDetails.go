package models

import "github.com/google/uuid"

type ChatDetails struct {
	ChatID       uuid.UUID `json:"chat_id"`
	ChatName     string    `json:"chat_name"`
	Messages     []Message `json:"messages"`
	Participants []User    `json:"participants"`
}
