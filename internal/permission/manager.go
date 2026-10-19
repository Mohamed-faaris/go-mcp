// Package permission enforces approval policies before tool execution.
package permission

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Mohamed-faaris/go-mcp/internal/logger"
)

// Mode controls which actions require approval.
type Mode string

const (
	ModeAll       Mode = "all"       // Every tool call requires approval.
	ModeDangerous Mode = "dangerous" // Only high-risk actions require approval.
	ModeNone      Mode = "none"      // No approvals (fully autonomous).
)

// ParseMode converts a string to a Mode.
func ParseMode(s string) Mode {
	switch strings.ToLower(s) {
	case "all":
		return ModeAll
	case "dangerous":
		return ModeDangerous
	case "none":
		return ModeNone
	default:
		return ModeAll
	}
}

// dangerousTools lists tool name patterns that are considered high-risk.
// The match is a substring check on the full tool name.
var dangerousTools = []string{
	"execute_command",
	"run_command",
	"shell",
	"write_file",
	"delete",
	"remove",
	"click",      // GUI actions
	"type_text",  // GUI actions
	"press_key",  // GUI actions
	"screenshot", // privacy
}

// Manager handles permission checks and user approval prompts.
type Manager struct {
	mode       Mode
	allowShell bool
	log        *logger.Logger
	reader     *bufio.Reader
}

// NewManager creates a permission manager.
func NewManager(mode string, allowShell bool, log *logger.Logger) *Manager {
	return &Manager{
		mode:       ParseMode(mode),
		allowShell: allowShell,
		log:        log,
		reader:     bufio.NewReader(os.Stdin),
	}
}

// Evaluate checks whether a tool call is allowed.
// It returns (allowed bool, reason string).
func (m *Manager) Evaluate(toolName string, args map[string]interface{}) (bool, string) {
	// Shell restriction check.
	if !m.allowShell && isShellTool(toolName) {
		return false, "shell execution is disabled (ALLOW_SHELL=false)"
	}

	switch m.mode {
	case ModeNone:
		return true, ""
	case ModeAll:
		return m.requestApproval(toolName, args)
	case ModeDangerous:
		if isDangerous(toolName) {
			return m.requestApproval(toolName, args)
		}
		return true, ""
	default:
		return m.requestApproval(toolName, args)
	}
}

// requestApproval prompts the user via stdin and waits for y/n.
func (m *Manager) requestApproval(toolName string, args map[string]interface{}) (bool, string) {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    APPROVAL REQUIRED                        ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Tool:  %-52s║\n", toolName)
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Arguments:                                                 ║")
	for k, v := range args {
		line := fmt.Sprintf("    %s: %v", k, v)
		if len(line) > 58 {
			line = line[:55] + "..."
		}
		fmt.Printf("║  %-60s║\n", line)
	}
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Approve? [y/N]:                                            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Print("> ")

	input, err := m.reader.ReadString('\n')
	if err != nil {
		m.log.Error("permission: read approval input", map[string]string{"error": err.Error()})
		return false, "failed to read approval input"
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "y" || input == "yes" {
		m.log.Info("permission: approved", map[string]string{"tool": toolName})
		return true, ""
	}
	m.log.Info("permission: denied", map[string]string{"tool": toolName})
	return false, "user denied the action"
}

func isDangerous(toolName string) bool {
	lower := strings.ToLower(toolName)
	for _, pattern := range dangerousTools {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

func isShellTool(toolName string) bool {
	lower := strings.ToLower(toolName)
	return strings.Contains(lower, "execute_command") ||
		strings.Contains(lower, "run_command") ||
		strings.Contains(lower, "shell")
}
