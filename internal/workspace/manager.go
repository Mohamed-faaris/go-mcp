// Package workspace creates and validates isolated host directories for task containers.
package workspace

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var taskIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

type Manager struct {
	root string
}

// NewManager creates the workspace root with private host permissions.
func NewManager(root string) (*Manager, error) {
	if root == "" {
		return nil, errors.New("workspace manager: root is required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("workspace manager: resolve root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("workspace manager: create root: %w", err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("workspace manager: resolve root symlinks: %w", err)
	}
	info, err := os.Stat(real)
	if err != nil {
		return nil, fmt.Errorf("workspace manager: stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("workspace manager: root must be a directory")
	}
	if err := os.Chmod(real, 0o700); err != nil {
		return nil, fmt.Errorf("workspace manager: restrict root permissions: %w", err)
	}
	return &Manager{root: real}, nil
}

func (m *Manager) Root() string { return m.root }

// Create makes a unique task directory. Repository cloning/snapshotting is a
// separate operation and should write only into this returned path.
func (m *Manager) Create(taskID string) (string, error) {
	if !taskIDPattern.MatchString(taskID) {
		return "", errors.New("workspace manager: invalid task ID")
	}
	suffix := make([]byte, 5)
	if _, err := rand.Read(suffix); err != nil {
		return "", fmt.Errorf("workspace manager: generate workspace ID: %w", err)
	}
	path := filepath.Join(m.root, taskID+"-"+hex.EncodeToString(suffix))
	if err := os.Mkdir(path, 0o700); err != nil {
		return "", fmt.Errorf("workspace manager: create task directory: %w", err)
	}
	validated, err := Validate(m.root, path)
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return validated, nil
}

func (m *Manager) Validate(path string) (string, error) {
	return Validate(m.root, path)
}

// Remove deletes only a validated workspace strictly below the managed root.
// The manager must ensure the task/container is stopped before calling Remove.
func (m *Manager) Remove(path string) error {
	validated, err := Validate(m.root, path)
	if err != nil {
		return err
	}
	if filepath.Dir(validated) != m.root {
		return errors.New("workspace manager: refusing to remove a non-task root")
	}
	if err := os.RemoveAll(validated); err != nil {
		return fmt.Errorf("workspace manager: remove task workspace: %w", err)
	}
	return nil
}
