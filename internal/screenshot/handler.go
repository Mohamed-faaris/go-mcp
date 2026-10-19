// Package screenshot captures desktop or application screenshots
// and encodes them for vision-capable LLMs.
package screenshot

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Mohamed-faaris/go-mcp/internal/logger"
	"github.com/Mohamed-faaris/go-mcp/internal/mcp"
)

// Handler captures and encodes screenshots via MCP tools.
type Handler struct {
	registry *mcp.Registry
	log      *logger.Logger
}

// NewHandler creates a screenshot handler.
func NewHandler(registry *mcp.Registry, log *logger.Logger) *Handler {
	return &Handler{
		registry: registry,
		log:      log,
	}
}

// CaptureResult holds an encoded screenshot.
type CaptureResult struct {
	Base64Data string // raw base64 data
	MimeType   string // e.g., "image/png"
	DataURI    string // full data URI for OpenAI vision
}

// Capture takes a screenshot using the first available screenshot tool.
// It searches across connected MCP servers for tools whose name contains
// "screenshot".
func (h *Handler) Capture(ctx context.Context) (*CaptureResult, error) {
	tools := h.registry.ListTools()
	var screenshotTool string
	for _, t := range tools {
		if containsAny(strings.ToLower(t.Name), "screenshot", "capture_screen", "take_screenshot") {
			screenshotTool = t.Name
			break
		}
	}
	if screenshotTool == "" {
		return nil, fmt.Errorf("screenshot: no screenshot tool found across connected MCP servers")
	}

	h.log.Debug("screenshot: capturing", map[string]string{"tool": screenshotTool})

	result, err := h.registry.CallTool(ctx, screenshotTool, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("screenshot: capture failed: %w", err)
	}

	if result.IsError {
		return nil, fmt.Errorf("screenshot: tool returned error")
	}

	// Find the image content block.
	for _, block := range result.Content {
		if block.Type == "image" && block.Data != "" {
			mime := block.MimeType
			if mime == "" {
				mime = "image/png"
			}
			return &CaptureResult{
				Base64Data: block.Data,
				MimeType:   mime,
				DataURI:    fmt.Sprintf("data:%s;base64,%s", mime, block.Data),
			}, nil
		}
	}

	// Some tools return screenshots as base64 text.
	for _, block := range result.Content {
		if block.Type == "text" && looksLikeBase64(block.Text) {
			return &CaptureResult{
				Base64Data: block.Text,
				MimeType:   "image/png",
				DataURI:    fmt.Sprintf("data:image/png;base64,%s", block.Text),
			}, nil
		}
	}

	return nil, fmt.Errorf("screenshot: no image data in tool response")
}

// EncodeFile reads a local image file and returns it base64-encoded.
func EncodeFile(path string) (string, error) {
	// We don't use os.ReadFile here to avoid importing os in this package
	// unnecessarily — the MCP-based capture path is the primary flow.
	// If local file encoding is needed, callers can use the mcp tools.
	return "", fmt.Errorf("screenshot: local file encoding not yet implemented (use MCP capture)")
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func looksLikeBase64(s string) bool {
	if len(s) < 100 {
		return false
	}
	// Quick heuristic: valid base64 chars and no spaces in first 100 chars.
	sample := s[:100]
	for _, c := range sample {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=') {
			return false
		}
	}
	// Verify it actually decodes.
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}
