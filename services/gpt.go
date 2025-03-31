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

func GetGPTResponse(messages []models.GPTMessage) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("API key is not set")
	}

	requestBody, err := json.Marshal(models.GPTRequest{
		Model:    "gpt-4o-mini",
		Messages: messages,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}

	var gptResponse models.GPTResponse
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(body, &gptResponse); err != nil {
		return "", err
	}

	if len(gptResponse.Choices) > 0 {
		return gptResponse.Choices[0].Message.Content, nil
	}

	return "", nil
}
