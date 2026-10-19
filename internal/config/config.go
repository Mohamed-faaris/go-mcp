// Package config loads and validates application configuration from
// environment variables and .env files.
package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime settings for the agent.
type Config struct {
	// LLM provider
	OpenAIBaseURL string
	OpenAIAPIKey  string
	OpenAIModel   string

	// MCP
	MCPConfigPath string

	// Execution limits
	MaxSteps              int
	MaxToolCalls          int
	RequestTimeoutSeconds int

	// Security
	ApprovalMode string // "all", "dangerous", "none"
	AllowShell   bool

	// Logging
	LogLevel string
	LogFile  string
}

// Load reads configuration from .env (if present) then environment variables.
// Environment variables always override .env values.
func Load() (*Config, error) {
	// Best-effort load of .env — not an error if missing.
	_ = loadDotenv(".env")

	cfg := &Config{
		OpenAIBaseURL:         getEnv("OPENAI_BASE_URL", ""),
		OpenAIAPIKey:          getEnv("OPENAI_API_KEY", ""),
		OpenAIModel:           getEnv("OPENAI_MODEL", ""),
		MCPConfigPath:         getEnv("MCP_CONFIG_PATH", "config/mcp_servers.json"),
		MaxSteps:              getEnvInt("MAX_STEPS", 12),
		MaxToolCalls:          getEnvInt("MAX_TOOL_CALLS", 24),
		RequestTimeoutSeconds: getEnvInt("REQUEST_TIMEOUT_SECONDS", 90),
		ApprovalMode:          getEnv("APPROVAL_MODE", "all"),
		AllowShell:            getEnvBool("ALLOW_SHELL", true),
		LogLevel:              getEnv("LOG_LEVEL", "INFO"),
		LogFile:               getEnv("LOG_FILE", "logs/agent.log"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate checks that required fields are present and values are sane.
func (c *Config) Validate() error {
	if c.OpenAIBaseURL == "" {
		return fmt.Errorf("config: OPENAI_BASE_URL is required")
	}
	if c.OpenAIAPIKey == "" {
		return fmt.Errorf("config: OPENAI_API_KEY is required")
	}
	if c.OpenAIModel == "" {
		return fmt.Errorf("config: OPENAI_MODEL is required")
	}
	switch c.ApprovalMode {
	case "all", "dangerous", "none":
		// valid
	default:
		return fmt.Errorf("config: APPROVAL_MODE must be all, dangerous, or none; got %q", c.ApprovalMode)
	}
	if c.MaxSteps < 1 {
		return fmt.Errorf("config: MAX_STEPS must be >= 1")
	}
	if c.MaxToolCalls < 1 {
		return fmt.Errorf("config: MAX_TOOL_CALLS must be >= 1")
	}
	return nil
}

// loadDotenv reads a .env file and sets values into the process environment.
// It does NOT override variables that are already set.
func loadDotenv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		// Strip surrounding quotes.
		val = strings.Trim(val, `"'`)
		// Don't override existing env vars.
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
	return scanner.Err()
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	s := getEnv(key, "")
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	s := getEnv(key, "")
	if s == "" {
		return fallback
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return fallback
	}
	return b
}
