package webconfig

import (
	"os"
	"testing"
)

func TestWebConfigLoad(t *testing.T) {
	yamlContent := `
server:
  port: 9090
  host: "127.0.0.1"

auth:
  mode: "basic"
  admin:
    username: "superadmin"
    password: "${TEST_ADMIN_PASS:-pass123}"

podman:
  socket: "/var/run/podman.sock"
`

	tmp, err := os.CreateTemp("", "webconfig-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(yamlContent); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	cfg, err := Load(tmp.Name())
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Auth.Admin.Password != "pass123" {
		t.Errorf("expected default pass123, got %s", cfg.Auth.Admin.Password)
	}

	// Now with environment variable set
	os.Setenv("TEST_ADMIN_PASS", "custom_secret")
	defer os.Unsetenv("TEST_ADMIN_PASS")

	cfg2, err := Load(tmp.Name())
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg2.Auth.Admin.Password != "custom_secret" {
		t.Errorf("expected custom_secret, got %s", cfg2.Auth.Admin.Password)
	}
}
