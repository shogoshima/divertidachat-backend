package models

import (
	"time"
)

// For communication with the frontend
type PublicProfile struct {
	ID          string     `json:"id"`
	DisplayName string     `json:"display_name"`
	Username    string     `json:"username"`
	PhotoURL    string     `json:"photo_url"`
	LastSeen    *time.Time `json:"last_seen"`
}
