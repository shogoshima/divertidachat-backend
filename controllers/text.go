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
		Name:    "Whatever...",
		Emoji:   "🙃",
		Command: "in a sarcastic way",
	},
	{
		ID:      5,
		Name:    "Glitch",
		Emoji:   "💻",
		Command: "with lots of typing errors",
	},
	{
		ID:      6,
		Name:    "Robot",
		Emoji:   "🤖",
		Command: "like a robot, with inputs and outputs",
	},
	{
		ID:      7,
		Name:    "Emoji",
		Emoji:   "😀",
		Command: "using only emojis",
	},
	{
		ID:      8,
		Name:    "Yoda",
		Emoji:   "👽",
		Command: "like Yoda",
	},
	{
		ID:      9,
		Name:    "Baby Talk",
		Emoji:   "🍼",
		Command: "like a baby trying to talk",
	},
	{
		ID:      10,
		Name:    "Valley Girl",
		Emoji:   "💅",
		Command: "like a stereotypical valley girl, saying 'like' a lot",
	},
	{
		ID:      11,
		Name:    "Grandma",
		Emoji:   "👵",
		Command: "like a sweet old grandma telling a story",
	},
	{
		ID:      12,
		Name:    "Caveman",
		Emoji:   "🪨",
		Command: "like a caveman with limited vocabulary",
	},
	{
		ID:      13,
		Name:    "Drama Queen",
		Emoji:   "🎭",
		Command: "in a dramatic and over-the-top way",
	},
	{
		ID:      14,
		Name:    "Detective Noir",
		Emoji:   "🕵️‍♂️",
		Command: "like a noir detective narrating a mystery",
	},
	{
		ID:      15,
		Name:    "Gamer",
		Emoji:   "🎮",
		Command: "like a gamer with slang and abbreviations",
	},
	{
		ID:      16,
		Name:    "Alien",
		Emoji:   "🛸",
		Command: "like an alien trying to understand humans",
	},
	{
		ID:      17,
		Name:    "Opera Singer",
		Emoji:   "🎤",
		Command: "like an opera singer, very dramatic and musical",
	},
	{
		ID:      18,
		Name:    "Medieval Peasant",
		Emoji:   "🛡️",
		Command: "like a medieval peasant from a fantasy world",
	},
	{
		ID:      19,
		Name:    "Corporate Jargon",
		Emoji:   "📊",
		Command: "using only corporate business jargon",
	},
	{
		ID:      20,
		Name:    "Karen",
		Emoji:   "🧍‍♀️",
		Command: "like a person demanding to speak to the manager",
	},
	{
		ID:      21,
		Name:    "Surfer Dude",
		Emoji:   "🏄",
		Command: "like a relaxed surfer dude",
	},
	{
		ID:      22,
		Name:    "Minion",
		Emoji:   "🍌",
		Command: "like a minion from the Despicable Me movies",
	},
	{
		ID:      23,
		Name:    "Text Message Teen",
		Emoji:   "📱",
		Command: "like a teenager texting with abbreviations and emojis",
	},
}

func GetTextFilters(c *gin.Context) {
	c.JSON(200, gin.H{"textfilters": TextFilters})
}
