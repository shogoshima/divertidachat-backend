package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/shogoshima/divertidachat-backend/models"
)

var TextFilters = []models.TextFilter{
	{
		ID:      0,
		Name:    "None",
		Emoji:   "🙄",
		Command: "",
	},
	{
		ID:      1,
		Name:    "Flirting",
		Emoji:   "💘",
		Command: "in a flirty manner.",
	},
	{
		ID:      2,
		Name:    "Pirate",
		Emoji:   "🏴‍☠️",
		Command: "like a pirate.",
	},
	{
		ID:      3,
		Name:    "Shakespeare",
		Emoji:   "🧙‍♂️",
		Command: "as if you were a Shakespearean character.",
	},
	{
		ID:      4,
		Name:    "Angry",
		Emoji:   "😡",
		Command: "as if you were angry.",
	},
	{
		ID:      5,
		Name:    "Glitch",
		Emoji:   "💻",
		Command: "with typing errors.",
	},
	{
		ID:      6,
		Name:    "Robot",
		Emoji:   "🤖",
		Command: "like a robot.",
	},
	{
		ID:      7,
		Name:    "Emoji",
		Emoji:   "😀",
		Command: "using only emojis.",
	},
	{
		ID:      8,
		Name:    "Yoda",
		Emoji:   "👽",
		Command: "like Yoda.",
	},
}

func GetTextFilters(c *gin.Context) {
	c.JSON(200, TextFilters)
}
