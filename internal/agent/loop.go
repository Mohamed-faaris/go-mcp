// Package agent implements the autonomous execution loop that ties
// together the LLM, MCP tools, permissions, and context management.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	agentctx "github.com/Mohamed-faaris/go-mcp/internal/context"
	"github.com/Mohamed-faaris/go-mcp/internal/llm"
	"github.com/Mohamed-faaris/go-mcp/internal/logger"
	"github.com/Mohamed-faaris/go-mcp/internal/mcp"
	"github.com/Mohamed-faaris/go-mcp/internal/permission"
	"github.com/Mohamed-faaris/go-mcp/internal/screenshot"
)

// Loop drives the autonomous execution cycle.
type Loop struct {
	llmClient  *llm.Client
	registry   *mcp.Registry
	ctxMgr     *agentctx.Manager
	permMgr    *permission.Manager
	screenshotH *screenshot.Handler
	log        *logger.Logger

	maxSteps     int
	maxToolCalls int
	toolDefs     []llm.ToolDef
}

// NewLoop creates an agent loop.
func NewLoop(
	llmClient *llm.Client,
	registry *mcp.Registry,
	ctxMgr *agentctx.Manager,
	permMgr *permission.Manager,
	screenshotH *screenshot.Handler,
	log *logger.Logger,
	maxSteps, maxToolCalls int,
) *Loop {
	// Build tool definitions from MCP servers.
	mcpTools := registry.ListTools()
	toolDefs := llm.ToolDefsFromMCP(mcpTools)

	log.Info("agent: initialized", map[string]interface{}{
		"tools_available": len(toolDefs),
		"max_steps":       maxSteps,
		"max_tool_calls":  maxToolCalls,
	})

	return &Loop{
		llmClient:    llmClient,
		registry:     registry,
		ctxMgr:       ctxMgr,
		permMgr:      permMgr,
		screenshotH:  screenshotH,
		log:          log,
		maxSteps:     maxSteps,
		maxToolCalls: maxToolCalls,
		toolDefs:     toolDefs,
	}
}

// RunTask executes a user task through the autonomous loop.
// It returns the final assistant response.
func (l *Loop) RunTask(ctx context.Context, task string) (string, error) {
	// Add the user message.
	l.ctxMgr.AddMessage(llm.Message{
		Role:    "user",
		Content: task,
	})

	totalSteps := 0
	totalToolCalls := 0

	for totalSteps < l.maxSteps {
		totalSteps++
		l.log.Info("agent: step", map[string]interface{}{
			"step":             totalSteps,
			"total_tool_calls": totalToolCalls,
		})

		// Call the LLM.
		choice, usage, err := l.llmClient.Complete(ctx, l.ctxMgr.Messages(), l.toolDefs)
		if err != nil {
			return "", fmt.Errorf("agent: LLM call failed at step %d: %w", totalSteps, err)
		}

		l.log.Debug("agent: llm response", map[string]interface{}{
			"finish_reason": choice.FinishReason,
			"usage":         usage,
		})

		// Add assistant message to context.
		l.ctxMgr.AddMessage(choice.Message)

		// If the model wants to call tools, process them.
		if len(choice.Message.ToolCalls) > 0 {
			for _, tc := range choice.Message.ToolCalls {
				if totalToolCalls >= l.maxToolCalls {
					l.log.Warn("agent: tool call limit reached", map[string]interface{}{
						"limit": l.maxToolCalls,
					})
					l.ctxMgr.AddMessage(llm.Message{
						Role:       "tool",
						ToolCallID: tc.ID,
						Content:    "Error: tool call limit reached. Please provide a final answer with the information gathered so far.",
					})
					continue
				}

				result := l.executeTool(ctx, tc)
				totalToolCalls++

				l.ctxMgr.AddMessage(llm.Message{
					Role:       "tool",
					ToolCallID: tc.ID,
					Content:    result,
				})
			}
			continue // Go back for the model to process tool results.
		}

		// No tool calls — the model is done.
		if choice.FinishReason == "stop" || choice.FinishReason == "end_turn" {
			content := extractContent(choice.Message)
			l.log.Info("agent: task complete", map[string]interface{}{
				"steps":      totalSteps,
				"tool_calls": totalToolCalls,
			})
			return content, nil
		}

		// If finish_reason is something else (length, etc.), try to continue.
		l.log.Warn("agent: unexpected finish_reason", map[string]string{
			"finish_reason": choice.FinishReason,
		})
	}

	return "", fmt.Errorf("agent: reached max steps (%d) without completing the task", l.maxSteps)
}

// executeTool runs a single tool call through permission checks and MCP execution.
func (l *Loop) executeTool(ctx context.Context, tc llm.ToolCall) string {
	toolName := tc.Function.Name
	l.log.Info("agent: tool call", map[string]string{
		"tool": toolName,
		"id":   tc.ID,
	})

	// Parse arguments.
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		l.log.Error("agent: parse tool args", map[string]string{
			"tool":  toolName,
			"error": err.Error(),
		})
		return fmt.Sprintf("Error: could not parse tool arguments: %s", err)
	}

	// Permission check.
	allowed, reason := l.permMgr.Evaluate(toolName, args)
	if !allowed {
		l.log.Warn("agent: tool denied", map[string]string{
			"tool":   toolName,
			"reason": reason,
		})
		return fmt.Sprintf("Permission denied: %s", reason)
	}

	// Execute via MCP.
	start := time.Now()
	result, err := l.registry.CallTool(ctx, toolName, args)
	duration := time.Since(start)

	l.log.LogAction(logger.ActionLog{
		Tool:      toolName,
		Arguments: args,
		Duration:  duration.String(),
	})

	if err != nil {
		l.log.Error("agent: tool execution failed", map[string]string{
			"tool":  toolName,
			"error": err.Error(),
		})
		return fmt.Sprintf("Error executing tool: %s", err)
	}

	if result.IsError {
		errText := extractTextContent(result)
		l.log.Error("agent: tool returned error", map[string]string{
			"tool":  toolName,
			"error": errText,
		})
		return fmt.Sprintf("Tool error: %s", errText)
	}

	// Format the result.
	return formatToolResult(result)
}

// formatToolResult converts a ToolResult into a string for the LLM context.
func formatToolResult(result *mcp.ToolResult) string {
	var parts []string
	for _, block := range result.Content {
		switch block.Type {
		case "text":
			parts = append(parts, block.Text)
		case "image":
			parts = append(parts, "[screenshot captured]")
		default:
			parts = append(parts, fmt.Sprintf("[%s content]", block.Type))
		}
	}
	if len(parts) == 0 {
		return "(no output)"
	}
	return strings.Join(parts, "\n")
}

func extractTextContent(result *mcp.ToolResult) string {
	for _, block := range result.Content {
		if block.Type == "text" {
			return block.Text
		}
	}
	return "(no text content)"
}

func extractContent(msg llm.Message) string {
	switch v := msg.Content.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		raw, _ := json.Marshal(v)
		return string(raw)
	}
}
