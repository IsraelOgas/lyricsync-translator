package translate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client is the LibreTranslate HTTP client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewClient creates a new LibreTranslate client.
func NewClient(baseURL string, timeoutSec int, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
		apiKey: apiKey,
	}
}

// translateRequest is the LibreTranslate API request body.
type translateRequest struct {
	Q      string `json:"q"`
	Source string `json:"source"`
	Target string `json:"target"`
	Format string `json:"format"`
}

// translateResponse is the LibreTranslate API response.
type translateResponse struct {
	TranslatedText string `json:"translatedText"`
}

// Translate sends text to LibreTranslate and returns the Spanish translation.
// source can be "auto" for automatic language detection.
func (c *Client) Translate(text, source, target string) (string, error) {
	if text == "" {
		return "", nil
	}

	body := translateRequest{
		Q:      text,
		Source: source,
		Target: target,
		Format: "text",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/translate", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("translate request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("libretranslate returned %d", resp.StatusCode)
	}

	var result translateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding translate response: %w", err)
	}

	return result.TranslatedText, nil
}
