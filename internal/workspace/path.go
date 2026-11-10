// Package workspace contains host-side path validation for manager-owned workspaces.
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrOutsideRoot = errors.New("workspace path is outside the configured workspace root")

// Validate resolves symlinks and requires an existing directory strictly below root.
func Validate(root, candidate string) (string, error) {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(candidate) == "" {
		return "", errors.New("workspace root and path are required")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	rootReal, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", fmt.Errorf("resolve root symlinks: %w", err)
	}
	candidateAbs, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve workspace: %w", err)
	}
	candidateReal, err := filepath.EvalSymlinks(candidateAbs)
	if err != nil {
		return "", fmt.Errorf("resolve workspace symlinks: %w", err)
	}
	info, err := os.Stat(candidateReal)
	if err != nil {
		return "", fmt.Errorf("stat workspace: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("workspace path must be a directory")
	}
	rel, err := filepath.Rel(rootReal, candidateReal)
	if err != nil {
		return "", fmt.Errorf("check workspace containment: %w", err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", ErrOutsideRoot
	}
	return candidateReal, nil
}
