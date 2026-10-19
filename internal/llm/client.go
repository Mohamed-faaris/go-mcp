// Package llm provides a client for OpenAI-compatible chat completion APIs
// with tool-calling support.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Mohamed-faaris/go-mcp/internal/logger"
	"github.com/Mohamed-faaris/go-mcp/internal/mcp"
)

// -----------------------------------------------------------------------
// OpenAI API types
// -----------------------------------------------------------------------

// Message is one entry in the conversation.
type Message struct {
	Role       string      `json:"role"`                  // system, user, assistant, tool
	Content    interface{} `json:"content,omitempty"`      // string or []ContentPart
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`   // assistant requesting tools
	ToolCallID string      `json:"tool_call_id,omitempty"` // for role=="tool"
	Name       string      `json:"name,omitempty"`         // tool name for role=="tool"
}

// ContentPart can be text or an image_url block inside a message.
type ContentPart struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`       // for type=="text"
	ImageURL *ImageURL `json:"image_url,omitempty"`  // for type=="image_url"
}

// ImageURL holds a base64 data URI or URL.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"` // "low", "high", "auto"
}

// ToolCall is an assistant's request to invoke a tool.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // always "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall holds the tool name and JSON-encoded arguments.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolDef describes a tool for the API.
type ToolDef struct {
	Type     string       `json:"type"` // "function"
	Function FunctionInfo `json:"function"`
}

// FunctionInfo is the schema of a function tool.
type FunctionInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// chatRequest is the body sent to /chat/completions.
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []ToolDef `json:"tools,omitempty"`
}

// chatResponse is the response from /chat/completions.
type chatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice is one completion result.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage tracks token counts.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// -----------------------------------------------------------------------
// Client
// -----------------------------------------------------------------------

// Client communicates with an OpenAI-compatible API.
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
	log        *logger.Logger
}

// NewClient creates an LLM client.
func NewClient(baseURL, apiKey, model string, timeoutSec int, log *logger.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
		log: log,
	}
}

// Complete sends a conversation to the model and returns the assistant message.
func (c *Client) Complete(ctx context.Context, messages []Message, tools []ToolDef) (*Choice, *Usage, error) {
	body := chatRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("llm: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, nil, fmt.Errorf("llm: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	c.log.Debug("llm: request", map[string]interface{}{
		"model":         c.model,
		"message_count": len(messages),
		"tool_count":    len(tools),
	})

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("llm: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("llm: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("llm: API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result chatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, nil, fmt.Errorf("llm: parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, nil, fmt.Errorf("llm: no choices returned")
	}

	c.log.Debug("llm: response", map[string]interface{}{
		"finish_reason": result.Choices[0].FinishReason,
		"usage":         result.Usage,
	})

	return &result.Choices[0], &result.Usage, nil
}

// ToolDefsFromMCP converts MCP tool schemas into OpenAI tool definitions.
func ToolDefsFromMCP(tools []mcp.ToolSchema) []ToolDef {
	defs := make([]ToolDef, len(tools))
	for i, t := range tools {
		params := t.InputSchema
		if params == nil {
			params = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}
		defs[i] = ToolDef{
			Type: "function",
			Function: FunctionInfo{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		}
	}
	return defs
}
