package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Mohamed-faaris/go-mcp/internal/webconfig"
)

// MCPRegistryTool is a reusable MCP server tool definition that can be
// attached to sandbox containers. Level controls visibility/attachability:
//   - "admin": only admins can see, create, or attach it
//   - "user":  available to every role
type MCPRegistryTool struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	Level       string   `json:"level"` // "admin" or "user"
	Description string   `json:"description"`
	BuiltIn     bool     `json:"builtIn"`
	CreatedBy   string   `json:"createdBy"`
	CreatedAt   string   `json:"createdAt"`
}

// mcpRegistry is a thread-safe, file-backed store of MCP tool definitions.
type mcpRegistry struct {
	mu    sync.Mutex
	path  string
	tools []MCPRegistryTool
}

// loadMCPRegistry loads the registry from disk and seeds built-in tools from
// the web config defaults on first run.
func loadMCPRegistry(cfg *webconfig.Config) *mcpRegistry {
	path := os.Getenv("MCP_REGISTRY_PATH")
	if path == "" {
		path = filepath.Join("data", "mcp_registry.json")
	}

	reg := &mcpRegistry{path: path}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &reg.tools)
	}

	// Seed built-ins (from config defaults) if not already present
	seeded := false
	for _, t := range cfg.Defaults.MCPTools {
		if reg.nameExistsLocked(t.Name, "") {
			continue
		}
		reg.tools = append(reg.tools, MCPRegistryTool{
			ID:        "builtin-" + t.Name,
			Name:      t.Name,
			Command:   t.Command,
			Args:      t.Args,
			Level:     "user",
			BuiltIn:   true,
			CreatedBy: "system",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
		seeded = true
	}
	if seeded {
		reg.mu.Lock()
		reg.saveLocked()
		reg.mu.Unlock()
	}
	return reg
}

// saveLocked writes the registry to disk atomically. Caller must hold the lock.
func (m *mcpRegistry) saveLocked() {
	data, err := json.MarshalIndent(m.tools, "", "  ")
	if err != nil {
		return
	}
	if dir := filepath.Dir(m.path); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	tmp := m.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, m.path)
}

func (m *mcpRegistry) list() []MCPRegistryTool {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MCPRegistryTool, len(m.tools))
	copy(out, m.tools)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (m *mcpRegistry) get(id string) (MCPRegistryTool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.tools {
		if t.ID == id {
			return t, true
		}
	}
	return MCPRegistryTool{}, false
}

func (m *mcpRegistry) nameExists(name, excludeID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.nameExistsLocked(name, excludeID)
}

func (m *mcpRegistry) nameExistsLocked(name, excludeID string) bool {
	for _, t := range m.tools {
		if t.ID != excludeID && strings.EqualFold(t.Name, name) {
			return true
		}
	}
	return false
}

func (m *mcpRegistry) create(tool MCPRegistryTool) MCPRegistryTool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tools = append(m.tools, tool)
	m.saveLocked()
	return tool
}

func (m *mcpRegistry) update(id string, mutate func(MCPRegistryTool) MCPRegistryTool) (MCPRegistryTool, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.tools {
		if t.ID == id {
			updated := mutate(t)
			m.tools[i] = updated
			m.saveLocked()
			return updated, true
		}
	}
	return MCPRegistryTool{}, false
}

func (m *mcpRegistry) delete(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.tools {
		if t.ID == id {
			m.tools = append(m.tools[:i], m.tools[i+1:]...)
			m.saveLocked()
			return true
		}
	}
	return false
}

func newMCPToolID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "mcp-" + hex.EncodeToString(b)
}

// --- Handlers ---------------------------------------------------------------

// handleListMCPTools returns tools visible to the calling role:
// admins see everything; viewers see user-level tools plus their own.
func (s *Server) handleListMCPTools(w http.ResponseWriter, r *http.Request) {
	role := getUserRole(r.Context())
	user := getUserName(r.Context())

	tools := []MCPRegistryTool{}
	for _, t := range s.registry.list() {
		if role == "admin" || t.Level != "admin" || t.CreatedBy == user {
			tools = append(tools, t)
		}
	}
	s.jsonOK(w, map[string]interface{}{"tools": tools})
}

// handleCreateMCPTool registers a new MCP tool. Viewers can create user-level
// tools; only admins can register admin-level tools.
func (s *Server) handleCreateMCPTool(w http.ResponseWriter, r *http.Request) {
	role := getUserRole(r.Context())
	user := getUserName(r.Context())

	var req struct {
		Name        string   `json:"name"`
		Command     string   `json:"command"`
		Args        []string `json:"args"`
		Level       string   `json:"level"`
		Description string   `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	command := strings.TrimSpace(req.Command)
	if name == "" || command == "" {
		s.jsonError(w, "name and command are required", http.StatusBadRequest)
		return
	}
	if strings.ContainsAny(name, "/\\") {
		s.jsonError(w, "name must not contain path separators", http.StatusBadRequest)
		return
	}

	level := strings.ToLower(strings.TrimSpace(req.Level))
	if level != "admin" && level != "user" {
		level = "user"
	}
	if role != "admin" {
		level = "user" // only admins may register admin-level tools
	}

	if s.registry.nameExists(name, "") {
		s.jsonError(w, "an MCP tool with this name already exists", http.StatusConflict)
		return
	}

	tool := s.registry.create(MCPRegistryTool{
		ID:          newMCPToolID(),
		Name:        name,
		Command:     command,
		Args:        req.Args,
		Level:       level,
		Description: strings.TrimSpace(req.Description),
		CreatedBy:   user,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	})
	s.jsonOK(w, map[string]interface{}{"tool": tool})
}

// handleUpdateMCPTool edits a tool. Admins can edit anything; creators can
// edit their own non-built-in tools (never escalating the level to admin).
func (s *Server) handleUpdateMCPTool(w http.ResponseWriter, r *http.Request) {
	role := getUserRole(r.Context())
	user := getUserName(r.Context())
	id := r.PathValue("id")

	tool, ok := s.registry.get(id)
	if !ok {
		s.jsonError(w, "MCP tool not found", http.StatusNotFound)
		return
	}
	if role != "admin" && (tool.CreatedBy != user || tool.BuiltIn) {
		s.jsonError(w, "Forbidden: cannot edit this tool", http.StatusForbidden)
		return
	}

	var req struct {
		Name        string   `json:"name"`
		Command     string   `json:"command"`
		Args        []string `json:"args"`
		Level       string   `json:"level"`
		Description string   `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(req.Name)
	command := strings.TrimSpace(req.Command)
	if name == "" || command == "" {
		s.jsonError(w, "name and command are required", http.StatusBadRequest)
		return
	}

	level := strings.ToLower(strings.TrimSpace(req.Level))
	if level != "admin" && level != "user" {
		level = tool.Level
	}
	if role != "admin" {
		level = "user" // no privilege escalation
	}

	for _, t := range s.registry.list() {
		if !strings.EqualFold(t.Name, name) || t.ID == id {
			continue
		}
		s.jsonError(w, "an MCP tool with this name already exists", http.StatusConflict)
		return
	}

	updated, found := s.registry.update(id, func(old MCPRegistryTool) MCPRegistryTool {
		old.Name = name
		old.Command = command
		old.Args = req.Args
		old.Level = level
		old.Description = strings.TrimSpace(req.Description)
		if role == "admin" {
			old.BuiltIn = false // an edited tool is now custom
		}
		return old
	})
	if !found {
		s.jsonError(w, "MCP tool not found", http.StatusNotFound)
		return
	}
	s.jsonOK(w, map[string]interface{}{"tool": updated})
}

// handleDeleteMCPTool removes a tool. Admins can delete anything; creators can
// delete their own non-built-in tools.
func (s *Server) handleDeleteMCPTool(w http.ResponseWriter, r *http.Request) {
	role := getUserRole(r.Context())
	user := getUserName(r.Context())
	id := r.PathValue("id")

	tool, ok := s.registry.get(id)
	if !ok {
		s.jsonError(w, "MCP tool not found", http.StatusNotFound)
		return
	}
	if role != "admin" && (tool.CreatedBy != user || tool.BuiltIn) {
		s.jsonError(w, "Forbidden: cannot delete this tool", http.StatusForbidden)
		return
	}

	s.registry.delete(id)
	s.jsonOK(w, map[string]string{"status": "deleted", "id": id})
}
