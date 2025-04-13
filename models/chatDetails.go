package models

import (
	"github.com/google/uuid"
)

type ChatDetails struct {
	ChatID       uuid.UUID        `json:"chat_id"`
	Messages     []Message        `json:"messages"`
	Participants []UserPublicInfo `json:"participants"`
	Page         int              `json:"page"`
	PageSize     int              `json:"page_size"`
}
