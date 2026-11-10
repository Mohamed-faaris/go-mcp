package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerCreatesPrivateTaskWorkspace(t *testing.T) {
	m, err := NewManager(filepath.Join(t.TempDir(), "workspaces"))
	if err != nil { t.Fatal(err) }
	path, err := m.Create("task-123")
	if err != nil { t.Fatal(err) }
	if _, err := os.Stat(path); err != nil { t.Fatal(err) }
	info, err := os.Stat(path)
	if err != nil { t.Fatal(err) }
	if info.Mode().Perm() != 0o700 { t.Fatalf("expected 0700, got %o", info.Mode().Perm()) }
}

func TestManagerRejectsUnsafeTaskID(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil { t.Fatal(err) }
	for _, id := range []string{"../escape", "/tmp/evil", "", "task/id"} {
		if _, err := m.Create(id); err == nil { t.Fatalf("expected invalid task ID %q to fail", id) }
	}
}

func TestManagerRemoveOnlyManagedChild(t *testing.T) {
	root := t.TempDir()
	m, err := NewManager(root)
	if err != nil { t.Fatal(err) }
	path, err := m.Create("task-1")
	if err != nil { t.Fatal(err) }
	if err := m.Remove(path); err != nil { t.Fatal(err) }
	if _, err := os.Stat(path); !os.IsNotExist(err) { t.Fatalf("workspace still exists: %v", err) }
	if err := m.Remove(root); err == nil { t.Fatal("manager must not remove its root") }
}
