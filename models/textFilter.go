package models

// For communicating with the frontend
type TextFilter struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Emoji   string `json:"emoji"`
	Command string `json:"command"`
}
