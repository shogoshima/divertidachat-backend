package models

import (
	"time"

	"github.com/google/uuid"
)

// For communication with the frontend
type UserPublicInfo struct {
	ID       uuid.UUID  `json:"id"`
	Name     string     `json:"name"`
	Username string     `json:"username"`
	PhotoURL string    `json:"photo_url"`
	LastSeen *time.Time `json:"last_seen"`
}
