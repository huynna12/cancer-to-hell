package gemma

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const apiBase = "https://generativelanguage.googleapis.com/v1beta/models"

var httpClient = &http.Client{Timeout: 5 * time.Minute}

// ── Request / Response types ──────────────────────────────────────────────────

type part struct {
	Text string `json:"text"`
}

type content struct {
	Role  string `json:"role,omitempty"`
	Parts []part `json:"parts"`
}

type generationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
	TopP            float64 `json:"topP"`
	TopK            int     `json:"topK"`
}

type generateRequest struct {
	SystemInstruction content          `json:"system_instruction"`
	Contents          []content        `json:"contents"`
	GenerationConfig  generationConfig `json:"generationConfig"`
}

type candidate struct {
	Content content `json:"content"`
}

type generateResponse struct {
	Candidates []candidate `json:"candidates"`
	Error      *apiError   `json:"error"`
}

type apiError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// ── Public API ────────────────────────────────────────────────────────────────

func Ask(systemPrompt string, userMessage string) (string, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GOOGLE_API_KEY is not set")
	}

	reqBody := buildGenerateRequest(systemPrompt, userMessage)
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", apiBase, modelName(), apiKey)
	body, err := postWithRetry(url, jsonBody)
	if err != nil {
		return "", err
	}

	return extractText(body)
}

func modelName() string {
	model := os.Getenv("GEMMA_MODEL")
	if model != "" {
		return model
	}
	return "gemma-4-e2b-it"
}

func buildGenerateRequest(systemPrompt string, userMessage string) generateRequest {
	return generateRequest{
		SystemInstruction: content{Parts: []part{{Text: systemPrompt}}},
		Contents:          []content{{Role: "user", Parts: []part{{Text: userMessage}}}},
		GenerationConfig: generationConfig{
			Temperature:     0.1,
			MaxOutputTokens: 768,
			TopP:            0.9,
			TopK:            40,
		},
	}
}

func postWithRetry(url string, jsonBody []byte) ([]byte, error) {
	const maxAttempts = 3

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		body, err := postAndRead(url, jsonBody)
		if err == nil {
			return body, nil
		}

		lastErr = err
		if attempt < maxAttempts {
			time.Sleep(3 * time.Second)
		}
	}

	return nil, lastErr
}

func postAndRead(url string, jsonBody []byte) ([]byte, error) {
	resp, err := httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("request to Google AI Studio failed: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func extractText(body []byte) (string, error) {
	var gemmaResp generateResponse
	if err := json.Unmarshal(body, &gemmaResp); err != nil {
		return "", fmt.Errorf("could not parse response: %s", err)
	}

	if gemmaResp.Error != nil {
		return "", fmt.Errorf("Google AI Studio error %d: %s", gemmaResp.Error.Code, gemmaResp.Error.Message)
	}

	if len(gemmaResp.Candidates) == 0 || len(gemmaResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from model")
	}

	return gemmaResp.Candidates[0].Content.Parts[0].Text, nil
}
