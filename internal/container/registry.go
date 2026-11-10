package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

var ErrRegistryNotFound = errors.New("container registry: record not found")

type Registry interface {
	Put(context.Context, Info) error
	Get(context.Context, string) (Info, error)
	Delete(context.Context, string) error
	List(context.Context) ([]Info, error)
}

// FileRegistry is an atomic JSON registry for a single control-plane writer.
// Use a transactional database implementation for multi-replica deployments.
type FileRegistry struct {
	mu sync.RWMutex
	path string
	records map[string]Info
}

func NewFileRegistry(path string) (*FileRegistry, error) {
	if path == "" {
		return nil, errors.New("container registry: path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("container registry: resolve path: %w", err)
	}
	r := &FileRegistry{path: abs, records: make(map[string]Info)}
	data, err := os.ReadFile(abs)
	if errors.Is(err, os.ErrNotExist) {
		return r, nil
	}
	if err != nil {
		return nil, fmt.Errorf("container registry: read: %w", err)
	}
	var records []Info
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("container registry: decode: %w", err)
	}
	for _, record := range records {
		if record.ID == "" || record.OwnerID == "" || record.TaskID == "" {
			return nil, errors.New("container registry: stored record has incomplete identity")
		}
		if _, exists := r.records[record.ID]; exists {
			return nil, fmt.Errorf("container registry: duplicate runtime ID %q", record.ID)
		}
		r.records[record.ID] = cloneInfo(record)
	}
	return r, nil
}

func (r *FileRegistry) Put(ctx context.Context, info Info) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if info.ID == "" || info.OwnerID == "" || info.TaskID == "" {
		return errors.New("container registry: runtime ID, owner ID and task ID are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[info.ID] = cloneInfo(info)
	return r.persistLocked()
}

func (r *FileRegistry) Get(ctx context.Context, id string) (Info, error) {
	if err := ctx.Err(); err != nil {
		return Info{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.records[id]
	if !ok {
		return Info{}, ErrRegistryNotFound
	}
	return cloneInfo(info), nil
}

func (r *FileRegistry) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.records[id]; !ok {
		return ErrRegistryNotFound
	}
	delete(r.records, id)
	return r.persistLocked()
}

func (r *FileRegistry) List(ctx context.Context) ([]Info, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Info, 0, len(r.records))
	for _, info := range r.records {
		out = append(out, cloneInfo(info))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *FileRegistry) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return fmt.Errorf("container registry: create directory: %w", err)
	}
	records := make([]Info, 0, len(r.records))
	for _, info := range r.records {
		records = append(records, cloneInfo(info))
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("container registry: encode: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(r.path), ".containers-*.tmp")
	if err != nil {
		return fmt.Errorf("container registry: create temp: %w", err)
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("container registry: write: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("container registry: sync: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("container registry: close: %w", err)
	}
	if err := os.Rename(tmp, r.path); err != nil {
		return fmt.Errorf("container registry: replace: %w", err)
	}
	return nil
}

func cloneInfo(in Info) Info {
	out := in
	if in.Labels != nil {
		out.Labels = make(map[string]string, len(in.Labels))
		for key, value := range in.Labels {
			out.Labels[key] = value
		}
	}
	return out
}
