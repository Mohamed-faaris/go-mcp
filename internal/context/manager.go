// Package context manages the conversation history (messages list)
// that is sent to the LLM on each turn.
package context

import (
	"github.com/Mohamed-faaris/go-mcp/internal/llm"
)

// Manager maintains the rolling conversation context.
type Manager struct {
	messages []llm.Message
	maxLen   int // max number of messages before trimming
}

// New creates a context manager. maxLen controls the maximum number
// of messages kept; older messages (except the system prompt) are
// dropped when this limit is reached.
func New(systemPrompt string, maxLen int) *Manager {
	m := &Manager{
		maxLen: maxLen,
	}
	if systemPrompt != "" {
		m.messages = append(m.messages, llm.Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}
	return m
}

// AddMessage appends a message to the conversation and trims if needed.
func (m *Manager) AddMessage(msg llm.Message) {
	m.messages = append(m.messages, msg)
	m.trim()
}

// Messages returns the full conversation for the next LLM call.
func (m *Manager) Messages() []llm.Message {
	out := make([]llm.Message, len(m.messages))
	copy(out, m.messages)
	return out
}

// Len returns the current number of messages.
func (m *Manager) Len() int {
	return len(m.messages)
}

// trim removes the oldest non-system messages when the conversation
// exceeds maxLen.
func (m *Manager) trim() {
	if m.maxLen <= 0 || len(m.messages) <= m.maxLen {
		return
	}

	// Keep the system prompt (index 0) and the newest maxLen-1 messages.
	keep := m.maxLen - 1
	if len(m.messages) > 0 && m.messages[0].Role == "system" {
		trimmed := make([]llm.Message, 0, m.maxLen)
		trimmed = append(trimmed, m.messages[0])
		trimmed = append(trimmed, m.messages[len(m.messages)-keep:]...)
		m.messages = trimmed
	} else {
		m.messages = m.messages[len(m.messages)-m.maxLen:]
	}
}
