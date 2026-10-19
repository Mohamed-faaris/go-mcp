// Package mcp implements a client for the Model Context Protocol over stdio.
// It manages server processes, discovers tools at runtime, and executes tool calls.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Mohamed-faaris/go-mcp/internal/logger"
)

// -----------------------------------------------------------------------
// JSON-RPC 2.0 types
// -----------------------------------------------------------------------

type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *jsonrpcError) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

// -----------------------------------------------------------------------
// MCP protocol types
// -----------------------------------------------------------------------

// ToolSchema describes a tool discovered from an MCP server.
type ToolSchema struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolResult is the return value of a tool call.
type ToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock is a piece of tool output (text or image).
type ContentBlock struct {
	Type     string `json:"type"`               // "text" or "image"
	Text     string `json:"text,omitempty"`      // for type=="text"
	Data     string `json:"data,omitempty"`      // base64 for type=="image"
	MimeType string `json:"mimeType,omitempty"`  // for type=="image"
}

// ServerConfig describes one MCP server entry from the config file.
type ServerConfig struct {
	Name        string   `json:"name"`
	Command     string   `json:"command"`
	Args        []string `json:"args"`
	Description string   `json:"description"`
}

// MCPConfigFile is the top-level structure of mcp_servers.json.
type MCPConfigFile struct {
	Servers []ServerConfig `json:"servers"`
}

// -----------------------------------------------------------------------
// Server connection
// -----------------------------------------------------------------------

// Server represents a running MCP server process.
type Server struct {
	Config ServerConfig
	Tools  []ToolSchema

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
	nextID atomic.Int64
	log    *logger.Logger
}

// -----------------------------------------------------------------------
// Registry
// -----------------------------------------------------------------------

// Registry manages connections to MCP servers.
type Registry struct {
	servers map[string]*Server
	log     *logger.Logger
	mu      sync.RWMutex
}

// NewRegistry creates an empty registry.
func NewRegistry(log *logger.Logger) *Registry {
	return &Registry{
		servers: make(map[string]*Server),
		log:     log,
	}
}

// LoadConfig reads the MCP server config file and connects to each server.
func (r *Registry) LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("mcp: read config: %w", err)
	}
	var cfg MCPConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("mcp: parse config: %w", err)
	}

	for _, sc := range cfg.Servers {
		if err := r.Connect(sc); err != nil {
			r.log.Error("mcp: failed to connect", map[string]string{
				"server": sc.Name,
				"error":  err.Error(),
			})
			// Continue connecting other servers — report but don't abort.
		}
	}
	return nil
}

// Connect starts a server process and performs the MCP handshake.
func (r *Registry) Connect(sc ServerConfig) error {
	r.log.Info("mcp: connecting", map[string]string{"server": sc.Name, "command": sc.Command})

	cmd := exec.Command(sc.Command, sc.Args...)
	cmd.Stderr = os.Stderr

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("mcp: stdin pipe for %s: %w", sc.Name, err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("mcp: stdout pipe for %s: %w", sc.Name, err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("mcp: start %s (%s): %w", sc.Name, sc.Command, err)
	}

	srv := &Server{
		Config: sc,
		cmd:    cmd,
		stdin:  stdinPipe,
		stdout: bufio.NewReader(stdoutPipe),
		log:    r.log,
	}

	// MCP initialization handshake.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	initResult, err := srv.call(ctx, "initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "go-mcp-agent",
			"version": "0.1.0",
		},
	})
	if err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("mcp: initialize %s: %w", sc.Name, err)
	}
	r.log.Debug("mcp: initialized", map[string]interface{}{
		"server": sc.Name,
		"result": string(initResult),
	})

	// Send initialized notification (no response expected).
	if err := srv.notify("notifications/initialized", nil); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("mcp: initialized notification %s: %w", sc.Name, err)
	}

	// Discover tools.
	toolsRaw, err := srv.call(ctx, "tools/list", nil)
	if err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("mcp: tools/list %s: %w", sc.Name, err)
	}
	var toolsResp struct {
		Tools []ToolSchema `json:"tools"`
	}
	if err := json.Unmarshal(toolsRaw, &toolsResp); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("mcp: parse tools %s: %w", sc.Name, err)
	}
	srv.Tools = toolsResp.Tools

	r.mu.Lock()
	r.servers[sc.Name] = srv
	r.mu.Unlock()

	r.log.Info("mcp: connected", map[string]interface{}{
		"server":     sc.Name,
		"tool_count": len(srv.Tools),
	})
	return nil
}

// ListTools returns all tools across all connected servers,
// prefixed with "server_name." for namespace clarity.
func (r *Registry) ListTools() []ToolSchema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var all []ToolSchema
	for name, srv := range r.servers {
		for _, t := range srv.Tools {
			tool := t
			tool.Name = name + "." + t.Name
			all = append(all, tool)
		}
	}
	return all
}

// CallTool executes a tool by its namespaced name (e.g., "desktop-commander.execute_command").
func (r *Registry) CallTool(ctx context.Context, namespacedName string, args map[string]interface{}) (*ToolResult, error) {
	serverName, toolName, ok := splitNamespacedTool(namespacedName)
	if !ok {
		return nil, fmt.Errorf("mcp: invalid tool name %q (expected server.tool)", namespacedName)
	}

	r.mu.RLock()
	srv, exists := r.servers[serverName]
	r.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("mcp: server %q not connected", serverName)
	}

	raw, err := srv.call(ctx, "tools/call", map[string]interface{}{
		"name":      toolName,
		"arguments": args,
	})
	if err != nil {
		return nil, err
	}

	var result ToolResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("mcp: parse tool result: %w", err)
	}
	return &result, nil
}

// Shutdown stops all server processes.
func (r *Registry) Shutdown() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, srv := range r.servers {
		r.log.Info("mcp: shutting down", map[string]string{"server": name})
		_ = srv.stdin.Close()
		_ = srv.cmd.Process.Kill()
		_ = srv.cmd.Wait()
	}
	r.servers = make(map[string]*Server)
}

// -----------------------------------------------------------------------
// Server I/O
// -----------------------------------------------------------------------

func (s *Server) call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := s.nextID.Add(1)
	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.send(req); err != nil {
		return nil, err
	}
	return s.receive(ctx, id)
}

func (s *Server) notify(method string, params interface{}) error {
	req := struct {
		JSONRPC string      `json:"jsonrpc"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params,omitempty"`
	}{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.send(req)
}

func (s *Server) send(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("mcp: marshal: %w", err)
	}
	data = append(data, '\n')
	_, err = s.stdin.Write(data)
	return err
}

func (s *Server) receive(ctx context.Context, expectedID int64) (json.RawMessage, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		line, err := s.stdout.ReadBytes('\n')
		if err != nil {
			return nil, fmt.Errorf("mcp: read: %w", err)
		}

		var resp jsonrpcResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			// Skip malformed lines (could be server log output).
			s.log.Debug("mcp: skipping non-JSON line", map[string]string{
				"line": string(line),
			})
			continue
		}

		// Skip notifications or responses to other requests.
		if resp.ID != expectedID {
			continue
		}

		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	}
}

// -----------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------

func splitNamespacedTool(name string) (server, tool string, ok bool) {
	for i := 0; i < len(name); i++ {
		if name[i] == '.' {
			return name[:i], name[i+1:], true
		}
	}
	return "", "", false
}
