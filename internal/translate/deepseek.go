package translate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// DeepSeekClient translates lyrics using DeepSeek's OpenAI-compatible /v1/chat/completions API.
type DeepSeekClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	model      string
}

// NewDeepSeekClient creates a new DeepSeek translation client.
func NewDeepSeekClient(apiKey, model, baseURL string, timeoutSec int) *DeepSeekClient {
	return &DeepSeekClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
		apiKey: apiKey,
		model:  model,
	}
}

// --- DeepSeek API types ---

type deepseekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepseekResponseFormat struct {
	Type string `json:"type"`
}

type deepseekChatRequest struct {
	Model          string                  `json:"model"`
	Messages       []deepseekMessage       `json:"messages"`
	Temperature    float64                 `json:"temperature,omitempty"`
	MaxTokens      int                     `json:"max_tokens,omitempty"`
	ResponseFormat *deepseekResponseFormat `json:"response_format,omitempty"`
}

type deepseekChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type deepseekBatchResponse struct {
	Romanized    []string `json:"romanized"`
	Translations []string `json:"translations"`
}

// TranslateBatch sends all lines to DeepSeek in a single chat completion request.
// Returns romanized (transliterated to ASCII) AND translated text.
func (c *DeepSeekClient) TranslateBatch(lines []string, source, target string) ([]string, []string, error) {
	if len(lines) == 0 {
		return nil, nil, nil
	}

	// Build numbered user prompt
	var sb strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&sb, "%d. %s\n", i+1, line)
	}

	n := len(lines)
	systemPrompt := fmt.Sprintf(
		"You are a song lyric translator. For each numbered line do two things:\n"+
			"1. Romanize it: transliterate non-Latin scripts (Japanese, Chinese, Korean, Cyrillic, etc.) to ASCII/Latin alphabet.\n"+
			"2. Translate it to %s preserving metaphor, tone, cultural references, and poetic flow.\n"+
			"Return ONLY a JSON object with \"romanized\" and \"translations\" arrays, each containing exactly %d strings. "+
			`Example: {"romanized": ["konnichiha", "sekai"], "translations": ["hola", "mundo"]}`,
		target, n,
	)

	userPrompt := fmt.Sprintf("Source=%s, Target=%s\n%s", source, target, sb.String())

	reqBody := deepseekChatRequest{
		Model: c.model,
		Messages: []deepseekMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature:    0.3,
		MaxTokens:      16384,
		ResponseFormat: &deepseekResponseFormat{Type: "json_object"},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling deepseek request: %w", err)
	}

	endpoint := strings.TrimRight(c.baseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, nil, fmt.Errorf("creating deepseek request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("deepseek request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("deepseek returned %d", resp.StatusCode)
	}

	var chatResp deepseekChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, nil, fmt.Errorf("decoding deepseek chat response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, nil, fmt.Errorf("deepseek returned no choices")
	}

	content := extractJSON(chatResp.Choices[0].Message.Content)

	var batchResp deepseekBatchResponse
	if err := json.Unmarshal([]byte(content), &batchResp); err != nil {
		return nil, nil, fmt.Errorf("decoding deepseek batch JSON: %w\nraw response: %s", err, content)
	}

	if len(batchResp.Romanized) != n || len(batchResp.Translations) != n {
		log.Printf("deepseek: size mismatch (expected %d, got romanized=%d translations=%d) — padding with empty strings",
			n, len(batchResp.Romanized), len(batchResp.Translations))
		batchResp.Romanized = padTo(batchResp.Romanized, n)
		batchResp.Translations = padTo(batchResp.Translations, n)
	}

	return batchResp.Romanized, batchResp.Translations, nil
}

// Translate delegates to TranslateBatch with a single-element slice.
func (c *DeepSeekClient) Translate(text, source, target string) (string, error) {
	_, translated, err := c.TranslateBatch([]string{text}, source, target)
	if err != nil {
		return "", err
	}
	if len(translated) == 0 {
		return "", nil
	}
	return translated[0], nil
}

// extractJSON strips markdown code fences and extracts raw JSON from LLM output.
// Handles: ```json\n{...}\n```, ```\n{...}\n```, or plain JSON text.
func extractJSON(content string) string {
	content = strings.TrimSpace(content)

	// Strip ```json or ``` fences
	if strings.HasPrefix(content, "```") {
		nl := strings.Index(content, "\n")
		if nl != -1 {
			content = content[nl+1:]
		}
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}
	}

	return strings.TrimSpace(content)
}

// padTo pads slice s to length n with empty strings. Returns s unchanged if len(s) >= n.
func padTo(s []string, n int) []string {
	for len(s) < n {
		s = append(s, "")
	}
	return s
}
