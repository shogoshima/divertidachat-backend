package models

import (
	"time"

	"github.com/google/uuid"
)

type ChatTimestamp struct {
	ChatID    uuid.UUID `json:"chat_id"`
	UpdatedAt time.Time `json:"updated_at"`
}
