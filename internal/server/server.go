package server

import (
	"fmt"
	"net/http"

	"github.com/Mohamed-faaris/go-mcp/internal/logger"
	"github.com/Mohamed-faaris/go-mcp/internal/podman"
	"github.com/Mohamed-faaris/go-mcp/internal/webconfig"
)

// Server coordinates HTTP routing, authorization, and Podman services.
type Server struct {
	cfg    *webconfig.Config
	podman *podman.Client
	log    *logger.Logger
	mux    *http.ServeMux
}

// New creates and configures a new Server instance.
func New(cfg *webconfig.Config, pc *podman.Client, log *logger.Logger) *Server {
	s := &Server{
		cfg:    cfg,
		podman: pc,
		log:    log,
		mux:    http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	// Authentication
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.mux.HandleFunc("GET /api/auth/me", s.withAuth(s.handleMe, "viewer"))

	// Containers - Read/Access (Viewer & Admin)
	s.mux.HandleFunc("GET /api/containers", s.withAuth(s.handleListContainers, "viewer"))
	s.mux.HandleFunc("GET /api/containers/{id}", s.withAuth(s.handleGetContainer, "viewer"))
	s.mux.HandleFunc("GET /api/containers/{id}/logs", s.withAuth(s.handleContainerLogs, "viewer"))
	s.mux.HandleFunc("GET /api/containers/{id}/terminal", s.withAuth(s.handleTerminal, "viewer"))

	// Containers - Management (Admin only)
	s.mux.HandleFunc("POST /api/containers", s.withAuth(s.handleCreateContainer, "admin"))
	s.mux.HandleFunc("POST /api/containers/{id}/start", s.withAuth(s.handleStartContainer, "admin"))
	s.mux.HandleFunc("POST /api/containers/{id}/stop", s.withAuth(s.handleStopContainer, "admin"))
	s.mux.HandleFunc("POST /api/containers/{id}/duplicate", s.withAuth(s.handleDuplicateContainer, "admin"))
	s.mux.HandleFunc("DELETE /api/containers/{id}", s.withAuth(s.handleDeleteContainer, "admin"))

	// Projects / Directories
	s.mux.HandleFunc("GET /api/projects", s.withAuth(s.handleListProjects, "viewer"))
	s.mux.HandleFunc("POST /api/projects/clone", s.withAuth(s.handleCloneProject, "admin"))

	// Images
	s.mux.HandleFunc("GET /api/images", s.withAuth(s.handleListImages, "viewer"))
	s.mux.HandleFunc("POST /api/images/pull", s.withAuth(s.handlePullImage, "admin"))

	// Devices (Host cameras & V4L2)
	s.mux.HandleFunc("GET /api/devices", s.withAuth(s.handleListDevices, "viewer"))

	// Configuration
	s.mux.HandleFunc("GET /api/config", s.withAuth(s.handleGetConfig, "viewer"))

	// SPA static serving
	s.mux.Handle("/", s.spaHandler("web/dist"))
}

// Start runs the HTTP listener on configured host:port.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	s.log.Info("web server listening", map[string]interface{}{
		"addr": addr,
		"port": s.cfg.Server.Port,
	})
	return http.ListenAndServe(addr, s.mux)
}
