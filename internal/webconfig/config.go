package webconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server      ServerConfig    `yaml:"server"`
	Auth        AuthConfig      `yaml:"auth"`
	Podman      PodmanConfig    `yaml:"podman"`
	Directories DirConfig       `yaml:"directories"`
	Defaults    DefaultsConfig  `yaml:"defaults"`
	Images      []ImageConfig   `yaml:"images"`
}

type ServerConfig struct {
	Port int    `yaml:"port"`
	Host string `yaml:"host"`
}

type AuthConfig struct {
	Mode   string    `yaml:"mode"` // "none", "basic"
	Admin  BasicAuth `yaml:"admin"`
	Viewer BasicAuth `yaml:"viewer"`
}

type BasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type PodmanConfig struct {
	Socket  string `yaml:"socket"`
	Network string `yaml:"network"`
}

type DirConfig struct {
	Projects   string `yaml:"projects"`
	TmpPodmans string `yaml:"tmp_podmans"`
}

type DefaultsConfig struct {
	BaseImage string          `yaml:"base_image"`
	MCPTools  []MCPToolConfig `yaml:"mcp_tools"`
	Resources ResourceConfig  `yaml:"resource_limits"`
}

type MCPToolConfig struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

type ResourceConfig struct {
	Memory string `yaml:"memory"`
	CPUs   string `yaml:"cpus"`
}

type ImageConfig struct {
	Name  string `yaml:"name"`
	Image string `yaml:"image"`
}

var envDefaultRegex = regexp.MustCompile(`\$\{([A-Za-z0-9_]+)(?::-([^}]*))?\}`)

func expandEnvWithDefaults(s string) string {
	return envDefaultRegex.ReplaceAllStringFunc(s, func(match string) string {
		submatches := envDefaultRegex.FindStringSubmatch(match)
		if len(submatches) >= 2 {
			key := submatches[1]
			defVal := ""
			if len(submatches) >= 3 {
				defVal = submatches[2]
			}
			if val, ok := os.LookupEnv(key); ok && val != "" {
				return val
			}
			return defVal
		}
		return match
	})
}

// Load loads and parses the web configuration file, expanding environment variables.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("webconfig: read %s: %w", path, err)
	}

	expanded := expandEnvWithDefaults(string(data))
	expanded = os.ExpandEnv(expanded)

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("webconfig: parse yaml: %w", err)
	}

	// Apply sensible defaults
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.Host == "" {
		cfg.Server.Host = "0.0.0.0"
	}
	if cfg.Podman.Socket == "" {
		cfg.Podman.Socket = "/run/podman/podman.sock"
	}

	// Auto-detect rootless Podman socket if default socket doesn't exist or is inaccessible
	if _, err := os.Stat(cfg.Podman.Socket); err != nil {
		if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
			userSock := filepath.Join(xdg, "podman", "podman.sock")
			if _, sErr := os.Stat(userSock); sErr == nil {
				cfg.Podman.Socket = userSock
			}
		}
	}
	if cfg.Auth.Mode == "" {
		cfg.Auth.Mode = "basic"
	}
	if cfg.Auth.Admin.Username == "" {
		cfg.Auth.Admin.Username = "admin"
	}
	if cfg.Auth.Admin.Password == "" {
		cfg.Auth.Admin.Password = "admin123"
	}
	if cfg.Auth.Viewer.Username == "" {
		cfg.Auth.Viewer.Username = "viewer"
	}
	if cfg.Auth.Viewer.Password == "" {
		cfg.Auth.Viewer.Password = "viewer123"
	}
	if cfg.Defaults.BaseImage == "" {
		cfg.Defaults.BaseImage = "ubuntu:24.04"
	}
	// Expand a leading ~ in directory paths to the current user's home dir
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		expandTilde := func(p string) string {
			if p == "~" {
				return home
			}
			if strings.HasPrefix(p, "~/") {
				return filepath.Join(home, strings.TrimPrefix(p, "~/"))
			}
			return p
		}
		cfg.Directories.Projects = expandTilde(cfg.Directories.Projects)
		cfg.Directories.TmpPodmans = expandTilde(cfg.Directories.TmpPodmans)
	}
	if strings.TrimSpace(cfg.Directories.Projects) == "" {
		cfg.Directories.Projects = "/host/projects"
	}

	return &cfg, nil
}
