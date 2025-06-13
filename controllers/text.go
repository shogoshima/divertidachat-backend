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
		Command: "in a flirty manner",
	},
	{
		ID:      2,
		Name:    "Pirate",
		Emoji:   "🏴‍☠️",
		Command: "like a pirate",
	},
	{
		ID:      3,
		Name:    "Shakespeare",
		Emoji:   "🧙‍♂️",
		Command: "as if you were a Shakespearean character",
	},
	{
		ID:      4,
		Name:    "Glitch",
		Emoji:   "💻",
		Command: "with lots of typing errors",
	},
	{
		ID:      5,
		Name:    "Emoji",
		Emoji:   "😀",
		Command: "using only emojis",
	},
	{
		ID:      6,
		Name:    "Yoda",
		Emoji:   "👽",
		Command: "like Yoda",
	},
	{
		ID:      7,
		Name:    "Valley Girl",
		Emoji:   "💅",
		Command: "like a stereotypical valley girl, saying 'like' a lot",
	},
	{
		ID:      8,
		Name:    "Drama Queen",
		Emoji:   "🎭",
		Command: "in a dramatic and over-the-top way",
	},
	{
		ID:      9,
		Name:    "Detective Noir",
		Emoji:   "🕵️‍♂️",
		Command: "like a noir detective narrating a mystery",
	},
	{
		ID:      10,
		Name:    "Corporate Jargon",
		Emoji:   "📊",
		Command: "using only corporate business jargon",
	},
	{
		ID:      11,
		Name:    "Karen",
		Emoji:   "🧍‍♀️",
		Command: "like a person demanding to speak to the manager",
	},
	{
		ID:      12,
		Name:    "Minion",
		Emoji:   "🍌",
		Command: "like a minion from the Despicable Me movies",
	},
	{
		ID:      13,
		Name:    "Zen Master",
		Emoji:   "🧘",
		Command: "like a calm zen master sharing ancient wisdom",
	},
	{
		ID: 14,
		Name: "Shrek",
		Emoji: "🧅",
		Command: "like Shrek, using ogre slang and swamp metaphors",
	},
}

func GetTextFilters(c *gin.Context) {
	c.JSON(200, gin.H{"textfilters": TextFilters})
}
