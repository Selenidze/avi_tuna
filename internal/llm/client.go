package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	EnvAPIToken = "LLM_API_TOKEN"
	EnvBaseURL  = "LLM_BASE_URL"
)

// Config holds LLM client configuration.
type Config struct {
	APIToken string
	BaseURL  string
}

// ConfigFromEnv reads LLM configuration from environment variables.
func ConfigFromEnv() (*Config, error) {
	token := os.Getenv(EnvAPIToken)
	if token == "" {
		return nil, fmt.Errorf("missing %s environment variable\n\nSet it with:\n export %s=your-api-token", EnvAPIToken, EnvAPIToken)
	}

	baseURL := os.Getenv(EnvBaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("missing %s environment variable\n\nSet it with:\n export %s=https://api.example.com/v1", EnvBaseURL, EnvBaseURL)
	}

	return &Config{
		APIToken: token,
		BaseURL:  baseURL,
	}, nil
}

// Client wraps the OpenAI-compatible client for LLM interactions.
type Client struct {
	httpClient *http.Client
	apiToken   string
	baseURL    string
}

// NewClient creates a new LLM client with the given configuration.
func NewClient(cfg *Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		apiToken:   cfg.APIToken,
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
	}
}

// ChatRequest holds parameters for a chat completion request.
type ChatRequest struct {
	Model          string
	SystemPrompt   string
	UserMessage    string
	Temperature    float64
	MaxTokens      int
	EnableThinking *bool
}

// ChatResponse holds the response from a chat completion.
type ChatResponse struct {
	Content      string
	Model        string // Resolved model name from API response
	ProviderURL  string // Provider base URL (set by Router)
	PromptTokens int
	OutputTokens int
	Duration     time.Duration // Request execution time (set by Router)
}

type chatTemplateKwargs struct {
	EnableThinking *bool `json:"enable_thinking,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model              string              `json:"model"`
	Messages           []chatMessage       `json:"messages"`
	Temperature        float32             `json:"temperature,omitempty"`
	MaxTokens          int                 `json:"max_tokens,omitempty"`
	ChatTemplateKwargs *chatTemplateKwargs `json:"chat_template_kwargs,omitempty"`
}

type chatCompletionResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Chat sends a chat completion request and returns the response.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	payload := chatCompletionRequest{
		Model: req.Model,
		Messages: []chatMessage{
			{Role: "system", Content: req.SystemPrompt},
			{Role: "user", Content: req.UserMessage},
		},
		Temperature: float32(req.Temperature),
		MaxTokens:   req.MaxTokens,
	}

	if req.EnableThinking != nil {
		payload.ChatTemplateKwargs = &chatTemplateKwargs{
			EnableThinking: req.EnableThinking,
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("chat completion failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var out chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned")
	}

	return &ChatResponse{
		Content:      out.Choices[0].Message.Content,
		Model:        out.Model,
		PromptTokens: out.Usage.PromptTokens,
		OutputTokens: out.Usage.CompletionTokens,
	}, nil
}
