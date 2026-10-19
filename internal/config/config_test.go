package config

import (
	"os"
	"testing"
)

func TestLoadDotenv(t *testing.T) {
	// Create a temp .env file.
	content := `
# Comment
OPENAI_BASE_URL=https://test.example/v1
OPENAI_API_KEY=test-key
OPENAI_MODEL=gpt-4
`
	f, err := os.CreateTemp("", "env-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	// Clear relevant env vars.
	for _, k := range []string{"OPENAI_BASE_URL", "OPENAI_API_KEY", "OPENAI_MODEL"} {
		os.Unsetenv(k)
	}

	if err := loadDotenv(f.Name()); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("OPENAI_BASE_URL"); got != "https://test.example/v1" {
		t.Errorf("OPENAI_BASE_URL = %q, want %q", got, "https://test.example/v1")
	}
}

func TestValidate(t *testing.T) {
	c := &Config{
		OpenAIBaseURL: "https://example.com/v1",
		OpenAIAPIKey:  "key",
		OpenAIModel:   "model",
		ApprovalMode:  "all",
		MaxSteps:      1,
		MaxToolCalls:  1,
	}
	if err := c.Validate(); err != nil {
		t.Errorf("valid config failed: %v", err)
	}

	c.ApprovalMode = "invalid"
	if err := c.Validate(); err == nil {
		t.Error("expected error for invalid approval mode")
	}
}
