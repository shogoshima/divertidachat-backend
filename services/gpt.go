package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/shogoshima/divertidachat-backend/models"
)

var apiKey = os.Getenv("OPENAI_API_KEY")

const apiUrl = "https://api.openai.com/v1/chat/completions"

func GetGPTResponse(messages []models.GPTMessage, senderId string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("OpenAI API key is not set")
	}

	user, err := fetchUser(senderId)
	if err != nil {
		return "", fmt.Errorf("failed to fetch user: %w", err)
	}

	if user.UsedTokens > 5000 {
		return "", fmt.Errorf("user has exceeded their token limit, try again tomorrow")
	}

	requestBody, err := json.Marshal(models.GPTRequest{
		Model:    "gpt-4o-mini",
		Messages: messages,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal GPT request: %w", err)
	}

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read OpenAI response: %w", err)
	}

	var gptResponse models.GPTResponse
	if err := json.Unmarshal(body, &gptResponse); err != nil {
		return "", fmt.Errorf("failed to unmarshal GPT response: %w", err)
	}

	if len(gptResponse.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from GPT")
	}

	newUsedTokens := user.UsedTokens + gptResponse.Usage.TotalTokens
	if err := updateUserTokenUsage(senderId, newUsedTokens); err != nil {
		return "", fmt.Errorf("failed to update user's token usage: %w", err)
	}

	return gptResponse.Choices[0].Message.Content, nil
}

// fetchUser retrieves the user from the DB
func fetchUser(id string) (*models.User, error) {
	var user models.User
	err := DB.Model(&user).Where("id = ?", id).First(&user).Error
	return &user, err
}

// updateUserTokenUsage updates the user's used token count
func updateUserTokenUsage(id string, tokens int) error {
	return DB.Model(&models.User{}).Where("id = ?", id).Update("used_tokens", tokens).Error
}
