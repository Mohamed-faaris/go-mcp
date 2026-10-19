package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Mohamed-faaris/go-mcp/internal/podman"
)

// Helper methods for JSON responses
func (s *Server) jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// handleLogin handles login check
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	role := ""
	if body.Username == s.cfg.Auth.Admin.Username && body.Password == s.cfg.Auth.Admin.Password {
		role = "admin"
	} else if body.Username == s.cfg.Auth.Viewer.Username && body.Password == s.cfg.Auth.Viewer.Password {
		role = "viewer"
	} else if strings.EqualFold(s.cfg.Auth.Mode, "none") {
		role = "admin"
	}

	if role == "" {
		s.jsonError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	s.jsonOK(w, map[string]interface{}{
		"username": body.Username,
		"role":     role,
	})
}

// handleMe returns details of the currently authenticated user
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	role := getUserRole(r.Context())
	username, _ := r.Context().Value(usernameContextKey).(string)
	s.jsonOK(w, map[string]interface{}{
		"username": username,
		"role":     role,
	})
}

// handleListContainers returns all containers managed by go-mcp
func (s *Server) handleListContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := s.podman.ListContainers(r.Context())
	if err != nil {
		s.jsonError(w, fmt.Sprintf("Failed to list containers: %v", err), http.StatusInternalServerError)
		return
	}
	if containers == nil {
		containers = []podman.ContainerSummary{}
	}
	s.jsonOK(w, containers)
}

// handleGetContainer returns full details of a container
func (s *Server) handleGetContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		s.jsonError(w, "Container ID required", http.StatusBadRequest)
		return
	}

	inspect, err := s.podman.InspectContainer(r.Context(), id)
	if err != nil {
		s.jsonError(w, fmt.Sprintf("Failed to inspect container: %v", err), http.StatusNotFound)
		return
	}
	s.jsonOK(w, inspect)
}

// handleCreateContainer creates and optionally starts a new container
func (s *Server) handleCreateContainer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string               `json:"name"`
		Image       string               `json:"image"`
		ProjectDir  string               `json:"projectDir"`
		MemoryLimit string               `json:"memoryLimit"`
		CPULimit    string               `json:"cpuLimit"`
		AutoStart   bool                 `json:"autoStart"`
		Env         []string             `json:"env"`
		MCPTools    []podman.MCPToolSpec `json:"mcpTools"`
		Devices     []string             `json:"devices"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		req.Name = fmt.Sprintf("agent-%d", os.Getpid())
	}
	if req.Image == "" {
		req.Image = s.cfg.Defaults.BaseImage
	}
	if len(req.MCPTools) == 0 {
		// Populate from defaults
		for _, tool := range s.cfg.Defaults.MCPTools {
			req.MCPTools = append(req.MCPTools, podman.MCPToolSpec{
				Name:    tool.Name,
				Command: tool.Command,
				Args:    tool.Args,
			})
		}
	}

	createOpts := podman.CreateContainerOpts{
		Name:        req.Name,
		Image:       req.Image,
		ProjectDir:  req.ProjectDir,
		MemoryLimit: req.MemoryLimit,
		CPULimit:    req.CPULimit,
		Env:         req.Env,
		MCPTools:    req.MCPTools,
		Devices:     req.Devices,
	}

	id, err := s.podman.CreateContainer(r.Context(), createOpts)
	if err != nil {
		s.jsonError(w, fmt.Sprintf("Failed to create container: %v", err), http.StatusInternalServerError)
		return
	}

	// Auto-start by default
	if req.AutoStart {
		if err := s.podman.StartContainer(r.Context(), id); err != nil {
			s.jsonError(w, fmt.Sprintf("Container created but failed to start: %v", err), http.StatusInternalServerError)
			return
		}
	}

	s.jsonOK(w, map[string]interface{}{
		"id":      id,
		"name":    req.Name,
		"status":  "created",
		"started": req.AutoStart,
	})
}

// handleStartContainer starts a container
func (s *Server) handleStartContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.podman.StartContainer(r.Context(), id); err != nil {
		s.jsonError(w, fmt.Sprintf("Start failed: %v", err), http.StatusInternalServerError)
		return
	}
	s.jsonOK(w, map[string]string{"status": "started"})
}

// handleStopContainer stops a container
func (s *Server) handleStopContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.podman.StopContainer(r.Context(), id); err != nil {
		s.jsonError(w, fmt.Sprintf("Stop failed: %v", err), http.StatusInternalServerError)
		return
	}
	s.jsonOK(w, map[string]string{"status": "stopped"})
}

// handleDeleteContainer removes a container
func (s *Server) handleDeleteContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.podman.RemoveContainer(r.Context(), id); err != nil {
		s.jsonError(w, fmt.Sprintf("Delete failed: %v", err), http.StatusInternalServerError)
		return
	}
	s.jsonOK(w, map[string]string{"status": "deleted"})
}

// handleDuplicateContainer clones an existing container
func (s *Server) handleDuplicateContainer(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		NewName string `json:"newName"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.NewName == "" {
		req.NewName = fmt.Sprintf("%s-clone", strings.TrimPrefix(id, "gomcp-"))
	}

	newID, err := s.podman.DuplicateContainer(r.Context(), id, req.NewName)
	if err != nil {
		s.jsonError(w, fmt.Sprintf("Duplicate failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.jsonOK(w, map[string]interface{}{
		"id":      newID,
		"newName": req.NewName,
		"status":  "duplicated",
	})
}

// handleContainerLogs returns logs for a container
func (s *Server) handleContainerLogs(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tailStr := r.URL.Query().Get("tail")
	tail := 200
	if tailStr != "" {
		if t, err := strconv.Atoi(tailStr); err == nil {
			tail = t
		}
	}

	logs, err := s.podman.GetLogs(r.Context(), id, tail)
	if err != nil {
		s.jsonError(w, fmt.Sprintf("Failed to get logs: %v", err), http.StatusInternalServerError)
		return
	}
	s.jsonOK(w, map[string]string{"logs": logs})
}

// projectSkipDirs are directory names excluded from project listings and picks.
var projectSkipDirs = map[string]bool{
	".git": true, "node_modules": true, "dist": true, "build": true,
	"target": true, ".venv": true, "venv": true, "__pycache__": true,
	".next": true, "vendor": true,
}

// handleListProjects lists directories in projects and tmp-podmans folders
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	type ProjectItem struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		Type     string `json:"type"` // "project" or "tmp"
		Modified string `json:"modified"`
	}

	var results []ProjectItem

	// Cap recursive listing so the picker stays usable on large project trees
	const maxProjectItems = 500

	scanDir := func(basePath, dirType string) {
		if basePath == "" {
			return
		}
		var walk func(dir string, depth int)
		walk = func(dir string, depth int) {
			if len(results) >= maxProjectItems {
				return
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				return
			}
			for _, e := range entries {
				if len(results) >= maxProjectItems {
					return
				}
				if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || projectSkipDirs[e.Name()] {
					continue
				}
				info, _ := e.Info()
				modTime := ""
				if info != nil {
					modTime = info.ModTime().Format("2006-01-02 15:04:05")
				}
				fullPath := filepath.Join(dir, e.Name())
				results = append(results, ProjectItem{
					Name:     e.Name(),
					Path:     fullPath,
					Type:     dirType,
					Modified: modTime,
				})
				// Recurse so any nested folder can be chosen (bounded depth)
				if depth < 4 {
					walk(fullPath, depth+1)
				}
			}
		}
		walk(basePath, 0)
	}

	scanDir(s.cfg.Directories.Projects, "project")
	scanDir(s.cfg.Directories.TmpPodmans, "tmp")

	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })

	s.jsonOK(w, map[string]interface{}{
		"projectsDir": s.cfg.Directories.Projects,
		"tmpDir":      s.cfg.Directories.TmpPodmans,
		"items":       results,
	})
}

// handleBrowseProjects returns the immediate child folders of a directory for
// the web UI folder picker. Paths are restricted to the configured projects
// and scratch (tmp-podmans) roots.
func (s *Server) handleBrowseProjects(w http.ResponseWriter, r *http.Request) {
	projectsRoot, _ := filepath.Abs(s.cfg.Directories.Projects)
	tmpRoot, _ := filepath.Abs(s.cfg.Directories.TmpPodmans)

	within := func(root, p string) bool {
		return root != "" && (p == root || strings.HasPrefix(p, root+string(filepath.Separator)))
	}

	q := strings.TrimSpace(r.URL.Query().Get("path"))
	if q == "" {
		q = projectsRoot
	}
	abs, err := filepath.Abs(q)
	if err != nil {
		s.jsonError(w, "Invalid path", http.StatusBadRequest)
		return
	}
	abs = filepath.Clean(abs)

	if !within(projectsRoot, abs) && !within(tmpRoot, abs) {
		s.jsonError(w, "path must be inside the projects or scratch directory", http.StatusBadRequest)
		return
	}

	rootType := "projects"
	if within(tmpRoot, abs) {
		rootType = "tmp"
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		s.jsonError(w, fmt.Sprintf("Cannot read directory: %v", err), http.StatusBadRequest)
		return
	}

	type browseItem struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		Type     string `json:"type"`
		Modified string `json:"modified"`
	}
	items := []browseItem{}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || projectSkipDirs[e.Name()] {
			continue
		}
		info, _ := e.Info()
		modTime := ""
		if info != nil {
			modTime = info.ModTime().Format("2006-01-02 15:04:05")
		}
		items = append(items, browseItem{
			Name:     e.Name(),
			Path:     filepath.Join(abs, e.Name()),
			Type:     rootType,
			Modified: modTime,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	// Parent for breadcrumb navigation (empty when at a root)
	parent := ""
	if dir := filepath.Dir(abs); within(projectsRoot, dir) || within(tmpRoot, dir) {
		parent = dir
	}

	s.jsonOK(w, map[string]interface{}{
		"path":   abs,
		"parent": parent,
		"root":   rootType,
		"items":  items,
	})
}

// handleCloneProject duplicates a directory to a new folder in projects or tmp
func (s *Server) handleCloneProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SourcePath string `json:"sourcePath"`
		TargetName string `json:"targetName"`
		Location   string `json:"location"` // "projects" or "tmp"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SourcePath == "" || req.TargetName == "" {
		s.jsonError(w, "sourcePath and targetName are required", http.StatusBadRequest)
		return
	}
	// Target name must be a plain folder name — no separators or traversal
	if filepath.Base(req.TargetName) != req.TargetName || strings.Contains(req.TargetName, "..") {
		s.jsonError(w, "targetName must be a plain folder name (no path separators)", http.StatusBadRequest)
		return
	}

	targetBase := s.cfg.Directories.Projects
	if req.Location == "tmp" {
		targetBase = s.cfg.Directories.TmpPodmans
	}

	targetDir := filepath.Join(targetBase, req.TargetName)

	// Reject overlapping source/target (e.g. cloning a folder into itself)
	srcAbs, _ := filepath.Abs(req.SourcePath)
	tgtAbs, _ := filepath.Abs(targetDir)
	sep := string(filepath.Separator)
	if tgtAbs == srcAbs || strings.HasPrefix(tgtAbs, srcAbs+sep) || strings.HasPrefix(srcAbs, tgtAbs+sep) {
		s.jsonError(w, "source and target must not overlap", http.StatusBadRequest)
		return
	}

	if err := copyDirectory(req.SourcePath, targetDir); err != nil {
		s.jsonError(w, fmt.Sprintf("Failed to clone directory: %v", err), http.StatusInternalServerError)
		return
	}

	s.jsonOK(w, map[string]string{
		"status":     "cloned",
		"targetPath": targetDir,
	})
}

// handleListImages returns configured and locally downloaded Podman images
func (s *Server) handleListImages(w http.ResponseWriter, r *http.Request) {
	podmanImages, _ := s.podman.ListImages(r.Context())
	s.jsonOK(w, map[string]interface{}{
		"configured": s.cfg.Images,
		"local":      podmanImages,
	})
}

// handlePullImage requests Podman to pull a new image
func (s *Server) handlePullImage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Reference == "" {
		s.jsonError(w, "Image reference is required", http.StatusBadRequest)
		return
	}

	if err := s.podman.PullImage(r.Context(), req.Reference); err != nil {
		s.jsonError(w, fmt.Sprintf("Failed to pull image: %v", err), http.StatusInternalServerError)
		return
	}

	s.jsonOK(w, map[string]string{
		"status":    "pulled",
		"reference": req.Reference,
	})
}

// handleListDevices scans host video/camera devices (V4L2)
func (s *Server) handleListDevices(w http.ResponseWriter, r *http.Request) {
	type DeviceInfo struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		Type     string `json:"type"` // "video" or "media"
		BusInfo  string `json:"busInfo,omitempty"`
	}

	var devices []DeviceInfo

	// Scan /sys/class/video4linux
	v4lDir := "/sys/class/video4linux"
	if entries, err := os.ReadDir(v4lDir); err == nil {
		for _, e := range entries {
			devName := e.Name()
			devPath := filepath.Join("/dev", devName)

			// Read device model / friendly name
			friendlyName := devName
			nameBytes, err := os.ReadFile(filepath.Join(v4lDir, devName, "name"))
			if err == nil {
				friendlyName = strings.TrimSpace(string(nameBytes))
			}

			// Read index
			indexStr := ""
			if idxBytes, err := os.ReadFile(filepath.Join(v4lDir, devName, "index")); err == nil {
				indexStr = strings.TrimSpace(string(idxBytes))
			}

			// Only report primary capture streams or distinctive streams
			busInfo := ""
			if indexStr != "" {
				busInfo = fmt.Sprintf("Stream #%s", indexStr)
			}

			devices = append(devices, DeviceInfo{
				Name:    friendlyName,
				Path:    devPath,
				Type:    "video",
				BusInfo: busInfo,
			})
		}
	}

	// Also scan /dev/media*
	if mediaFiles, err := filepath.Glob("/dev/media*"); err == nil {
		for _, m := range mediaFiles {
			devices = append(devices, DeviceInfo{
				Name: fmt.Sprintf("Media Controller (%s)", filepath.Base(m)),
				Path: m,
				Type: "media",
			})
		}
	}

	s.jsonOK(w, devices)
}

// handleGetConfig returns public configuration (snake_case keys for the web UI)
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	type imageOut struct {
		Name  string `json:"name"`
		Image string `json:"image"`
	}
	images := make([]imageOut, 0, len(s.cfg.Images))
	for _, img := range s.cfg.Images {
		images = append(images, imageOut{Name: img.Name, Image: img.Image})
	}

	tools := make([]map[string]interface{}, 0, len(s.cfg.Defaults.MCPTools))
	for _, t := range s.cfg.Defaults.MCPTools {
		tools = append(tools, map[string]interface{}{
			"name":    t.Name,
			"command": t.Command,
			"args":    t.Args,
		})
	}

	s.jsonOK(w, map[string]interface{}{
		"defaults": map[string]interface{}{
			"base_image": s.cfg.Defaults.BaseImage,
			"resource_limits": map[string]string{
				"memory": s.cfg.Defaults.Resources.Memory,
				"cpus":   s.cfg.Defaults.Resources.CPUs,
			},
			"mcp_tools": tools,
		},
		"directories": map[string]string{
			"projects":    s.cfg.Directories.Projects,
			"tmp_podmans": s.cfg.Directories.TmpPodmans,
		},
		"images":  images,
		"authMode": s.cfg.Auth.Mode,
	})
}

// Helper: copyDirectory recursively copies a directory
func copyDirectory(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		// Recreate symlinks as-is (e.g. .venv/lib64 -> lib)
		if entry.Type()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(srcPath)
			if err != nil {
				return err
			}
			_ = os.Remove(dstPath)
			if err := os.Symlink(linkTarget, dstPath); err != nil {
				return err
			}
			continue
		}
		if entry.IsDir() {
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else if !entry.Type().IsRegular() {
			// Skip sockets, devices, fifos, etc.
			continue
		} else {
			in, err := os.Open(srcPath)
			if err != nil {
				return err
			}
			out, err := os.Create(dstPath)
			if err != nil {
				in.Close()
				return err
			}
			_, copyErr := io.Copy(out, in)
			in.Close()
			out.Close()
			if copyErr != nil {
				return copyErr
			}
		}
	}
	return nil
}
