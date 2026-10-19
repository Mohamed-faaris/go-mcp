package podman

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client interacts with the local Podman service over Unix domain socket.
type Client struct {
	socketPath string
	httpClient *http.Client
	baseURL    string
}

// NewClient returns a new Podman client connecting to the given Unix socket.
func NewClient(socketPath string) *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		},
		DisableKeepAlives: true,
	}

	return &Client{
		socketPath: socketPath,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   60 * time.Second,
		},
		baseURL: "http://d/v4.0.0/libpod",
	}
}

// Ping verifies connection to the Podman service.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://d/v4.0.0/libpod/_ping", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("podman ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("podman ping returned status %d", resp.StatusCode)
	}
	return nil
}

// ListContainers returns all containers matching prefix "gomcp-" or all managed containers.
func (c *Client) ListContainers(ctx context.Context) ([]ContainerSummary, error) {
	u := fmt.Sprintf("%s/containers/json?all=true", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list containers request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list containers error (%d): %s", resp.StatusCode, string(body))
	}

	var all []ContainerSummary
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		return nil, fmt.Errorf("failed to decode containers list: %w", err)
	}

	// Filter for containers managed by go-mcp or prefixed with gomcp-
	var filtered []ContainerSummary
	for _, item := range all {
		isManaged := false
		if item.Labels != nil && item.Labels["gomcp.managed"] == "true" {
			isManaged = true
		}
		for _, name := range item.Names {
			cleanName := strings.TrimPrefix(name, "/")
			if strings.HasPrefix(cleanName, "gomcp-") {
				isManaged = true
				break
			}
		}
		if isManaged {
			if item.Labels != nil {
				item.ProjectDir = item.Labels["gomcp.project"]
			}
			filtered = append(filtered, item)
		}
	}

	return filtered, nil
}

// InspectContainer gets full details for a given container ID or name.
func (c *Client) InspectContainer(ctx context.Context, idOrName string) (*ContainerInspect, error) {
	u := fmt.Sprintf("%s/containers/%s/json", c.baseURL, url.PathEscape(idOrName))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("inspect container %s failed (%d): %s", idOrName, resp.StatusCode, string(body))
	}

	var inspect ContainerInspect
	if err := json.NewDecoder(resp.Body).Decode(&inspect); err != nil {
		return nil, err
	}
	return &inspect, nil
}

// parseMemoryLimit converts human-readable strings like "2g", "512m" to bytes.
func parseMemoryLimit(mem string) int64 {
	mem = strings.TrimSpace(strings.ToLower(mem))
	if mem == "" {
		return 2 * 1024 * 1024 * 1024 // 2GB default
	}
	multipliers := map[string]int64{
		"k": 1024,
		"m": 1024 * 1024,
		"g": 1024 * 1024 * 1024,
	}
	for suffix, mult := range multipliers {
		if strings.HasSuffix(mem, suffix) {
			valStr := strings.TrimSuffix(mem, suffix)
			val, err := strconv.ParseInt(valStr, 10, 64)
			if err == nil {
				return val * mult
			}
		}
	}
	if val, err := strconv.ParseInt(mem, 10, 64); err == nil {
		return val
	}
	return 2 * 1024 * 1024 * 1024
}

// parseCPULimit converts CPU string like "2" or "1.5" to quota & period.
func parseCPULimit(cpuStr string) (quota int64, period uint64) {
	period = 100000 // 100ms
	cpuStr = strings.TrimSpace(cpuStr)
	if cpuStr == "" {
		return 200000, period // 2 cores
	}
	cpuVal, err := strconv.ParseFloat(cpuStr, 64)
	if err != nil || cpuVal <= 0 {
		return 200000, period
	}
	quota = int64(cpuVal * float64(period))
	return quota, period
}

// CreateContainer creates a new agent sandbox container in Podman.
func (c *Client) CreateContainer(ctx context.Context, opts CreateContainerOpts) (string, error) {
	name := strings.TrimSpace(opts.Name)
	if !strings.HasPrefix(name, "gomcp-") {
		name = "gomcp-" + name
	}
	if opts.Image == "" {
		opts.Image = "ubuntu:24.04"
	}

	labels := opts.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["gomcp.managed"] = "true"
	labels["gomcp.created_at"] = time.Now().UTC().Format(time.RFC3339)
	if opts.ProjectDir != "" {
		labels["gomcp.project"] = opts.ProjectDir
	}

	// Prepare MCP tools configuration into an environment variable or metadata
	toolsJSON, _ := json.Marshal(opts.MCPTools)
	labels["gomcp.mcp_tools"] = string(toolsJSON)

	// Mounts
	mounts := []map[string]interface{}{}
	if opts.ProjectDir != "" {
		mounts = append(mounts, map[string]interface{}{
			"Destination": "/workspace",
			"Source":      opts.ProjectDir,
			"Type":        "bind",
			"Options":     []string{"rbind", "rw"},
		})
	}

	memBytes := parseMemoryLimit(opts.MemoryLimit)
	cpuQuota, cpuPeriod := parseCPULimit(opts.CPULimit)

	envPairs := append(opts.Env,
		"GOMCP_CONTAINER=true",
		"WORKSPACE=/workspace",
		fmt.Sprintf("GOMCP_MCP_SERVERS=%s", string(toolsJSON)),
	)
	// Podman's SpecGenerator.env is a map[string]string (KEY -> value), not a slice
	env := make(map[string]string, len(envPairs))
	for _, kv := range envPairs {
		if k, v, ok := strings.Cut(kv, "="); ok && k != "" {
			env[k] = v
		}
	}

	payload := map[string]interface{}{
		"name":            name,
		"image":           opts.Image,
		"command":         []string{"/bin/sh", "-c", "while true; do sleep 3600; done"},
		"terminal":        true,
		"stdin":           true,
		"labels":          labels,
		"mounts":          mounts,
		"env":             env,
		"work_dir":        "/workspace",
		"resource_limits": map[string]interface{}{
			"memory": map[string]interface{}{
				"limit": memBytes,
			},
			"cpu": map[string]interface{}{
				"quota":  cpuQuota,
				"period": cpuPeriod,
			},
		},
	}

	var devices []map[string]interface{}
	for _, dev := range opts.Devices {
		dev = strings.TrimSpace(dev)
		if dev != "" {
			devices = append(devices, map[string]interface{}{
				"path": dev,
			})
		}
	}
	if len(devices) > 0 {
		payload["devices"] = devices
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	u := fmt.Sprintf("%s/containers/create", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("create container request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create container failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var res struct {
		ID       string   `json:"Id"`
		Warnings []string `json:"Warnings"`
	}
	if err := json.Unmarshal(respBody, &res); err != nil {
		return "", fmt.Errorf("decode create response: %w", err)
	}

	return res.ID, nil
}

// StartContainer starts a created or stopped container.
func (c *Client) StartContainer(ctx context.Context, idOrName string) error {
	u := fmt.Sprintf("%s/containers/%s/start", c.baseURL, url.PathEscape(idOrName))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("start container %s failed (%d): %s", idOrName, resp.StatusCode, string(body))
	}
	return nil
}

// StopContainer stops a running container.
func (c *Client) StopContainer(ctx context.Context, idOrName string) error {
	u := fmt.Sprintf("%s/containers/%s/stop", c.baseURL, url.PathEscape(idOrName))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotModified {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stop container %s failed (%d): %s", idOrName, resp.StatusCode, string(body))
	}
	return nil
}

// RemoveContainer force-removes a container.
func (c *Client) RemoveContainer(ctx context.Context, idOrName string) error {
	u := fmt.Sprintf("%s/containers/%s?force=true", c.baseURL, url.PathEscape(idOrName))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remove container %s failed (%d): %s", idOrName, resp.StatusCode, string(body))
	}
	return nil
}

// GetLogs returns the logs of a container.
func (c *Client) GetLogs(ctx context.Context, idOrName string, tail int) (string, error) {
	if tail <= 0 {
		tail = 200
	}
	u := fmt.Sprintf("%s/containers/%s/logs?stdout=true&stderr=true&tail=%d", c.baseURL, url.PathEscape(idOrName), tail)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// DuplicateContainer creates and starts a clone of an existing container with a new name.
func (c *Client) DuplicateContainer(ctx context.Context, sourceIDOrName, newName string) (string, error) {
	source, err := c.InspectContainer(ctx, sourceIDOrName)
	if err != nil {
		return "", fmt.Errorf("inspect source container: %w", err)
	}

	projectDir := ""
	for _, m := range source.Mounts {
		if m.Destination == "/workspace" {
			projectDir = m.Source
			break
		}
	}

	var tools []MCPToolSpec
	if source.Config.Labels != nil && source.Config.Labels["gomcp.mcp_tools"] != "" {
		_ = json.Unmarshal([]byte(source.Config.Labels["gomcp.mcp_tools"]), &tools)
	}

	createOpts := CreateContainerOpts{
		Name:        newName,
		Image:       source.Config.Image,
		ProjectDir:  projectDir,
		MCPTools:    tools,
		MemoryLimit: "2g",
		CPULimit:    "2",
	}

	newID, err := c.CreateContainer(ctx, createOpts)
	if err != nil {
		return "", fmt.Errorf("create duplicate: %w", err)
	}

	if err := c.StartContainer(ctx, newID); err != nil {
		return newID, fmt.Errorf("start duplicated container: %w", err)
	}

	return newID, nil
}

// ListImages lists container images available locally.
func (c *Client) ListImages(ctx context.Context) ([]ImageSummary, error) {
	u := fmt.Sprintf("%s/images/json", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var images []ImageSummary
	if err := json.NewDecoder(resp.Body).Decode(&images); err != nil {
		return nil, err
	}
	return images, nil
}

// PullImage requests Podman to pull a container image.
func (c *Client) PullImage(ctx context.Context, reference string) error {
	u := fmt.Sprintf("%s/images/pull?reference=%s", c.baseURL, url.QueryEscape(reference))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull image failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// CreateExec initializes an interactive execution session (bash or sh).
func (c *Client) CreateExec(ctx context.Context, idOrName string, cmd []string) (string, error) {
	if len(cmd) == 0 {
		cmd = []string{"/bin/bash"}
	}

	payload := ExecCreateConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          cmd,
	}

	data, _ := json.Marshal(payload)
	u := fmt.Sprintf("%s/containers/%s/exec", c.baseURL, url.PathEscape(idOrName))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create exec failed (%d): %s", resp.StatusCode, string(body))
	}

	var res ExecCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.ID, nil
}

// DialUnixConn dials raw connection to the Podman Unix socket for bi-directional exec streaming.
func (c *Client) DialUnixConn() (net.Conn, error) {
	return net.Dial("unix", c.socketPath)
}
