package gemma

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const ollamaURL = "http://localhost:11434/api/generate"

type Request struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type Response struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func Ask(systemPrompt string, userMessage string) (string, error) {
	fullPrompt := fmt.Sprintf("%s\n\n%s", systemPrompt, userMessage)

	reqBody := Request{
		Model:  "gemma4",
		Prompt: fullPrompt,
		Stream: false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(ollamaURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to connect to Ollama: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var gemmaResp Response
	if err := json.Unmarshal(body, &gemmaResp); err != nil {
		return "", err
	}

	if gemmaResp.Response == "" {
		return "", fmt.Errorf("empty response from Gemma")
	}

	return gemmaResp.Response, nil
}
