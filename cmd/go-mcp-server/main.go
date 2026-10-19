package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Mohamed-faaris/go-mcp/internal/logger"
	"github.com/Mohamed-faaris/go-mcp/internal/podman"
	"github.com/Mohamed-faaris/go-mcp/internal/server"
	"github.com/Mohamed-faaris/go-mcp/internal/webconfig"
)

func main() {
	configPath := flag.String("config", "config/web_config.yaml", "Path to web configuration file")
	flag.Parse()

	cfg, err := webconfig.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading web config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New("INFO", "logs/server.log")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	log.Info("go-mcp-server starting", map[string]interface{}{
		"port":          cfg.Server.Port,
		"auth_mode":     cfg.Auth.Mode,
		"podman_socket": cfg.Podman.Socket,
	})

	// Initialize Podman socket client
	podmanClient := podman.NewClient(cfg.Podman.Socket)

	// Quick check of Podman socket (non-fatal warning if not running yet)
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := podmanClient.Ping(pingCtx); err != nil {
		log.Warn("podman socket check failed (is podman.sock active?)", map[string]string{
			"socket": cfg.Podman.Socket,
			"error":  err.Error(),
		})
		fmt.Fprintf(os.Stderr, "Warning: Unable to reach Podman at %s: %v\n", cfg.Podman.Socket, err)
		fmt.Fprintf(os.Stderr, "Hint: Start the podman socket via: systemctl --user start podman.socket\n")
	} else {
		log.Info("podman socket connection verified", map[string]string{
			"socket": cfg.Podman.Socket,
		})
	}
	pingCancel()

	srv := server.New(cfg, podmanClient, log)

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Error("server listener error", map[string]string{"error": err.Error()})
		}
	}()

	fmt.Printf("🚀 go-mcp Web UI listening on http://%s:%d\n", cfg.Server.Host, cfg.Server.Port)
	<-stop
	fmt.Println("\nShutting down server gracefully...")
}
