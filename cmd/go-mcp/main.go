// go-mcp is an autonomous AI agent that controls an Ubuntu computer
// through MCP (Model Context Protocol) servers, using an OpenAI-compatible API.
//
// Usage:
//
//	go-mcp "open firefox and search for golang tutorials"
//	go-mcp --check  # verify MCP server connections and list tools
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Mohamed-faaris/go-mcp/internal/agent"
	agentctx "github.com/Mohamed-faaris/go-mcp/internal/context"
	"github.com/Mohamed-faaris/go-mcp/internal/config"
	"github.com/Mohamed-faaris/go-mcp/internal/llm"
	"github.com/Mohamed-faaris/go-mcp/internal/logger"
	"github.com/Mohamed-faaris/go-mcp/internal/mcp"
	"github.com/Mohamed-faaris/go-mcp/internal/permission"
	"github.com/Mohamed-faaris/go-mcp/internal/screenshot"
)

const systemPrompt = `You are an autonomous AI agent controlling an Ubuntu computer.
You have access to tools for terminal commands, browser automation, and desktop GUI control.

Guidelines:
- Use structured browser tools (Playwright) whenever possible.
- Fall back to screenshot-based GUI control (Ubuntu GUI MCP) only when browser APIs cannot reach the target.
- Always verify tool results before proceeding. Never assume an action succeeded.
- If a tool fails, try an alternative approach before giving up.
- When done, provide a clear summary of what was accomplished.
`

func main() {
	checkFlag := flag.Bool("check", false, "verify MCP connections and list available tools")
	flag.Parse()

	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Hint: copy .env.example to .env and fill in your values.\n")
		os.Exit(1)
	}

	// Set up logger.
	log, err := logger.New(cfg.LogLevel, cfg.LogFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	log.Info("go-mcp starting", map[string]string{
		"model":         cfg.OpenAIModel,
		"approval_mode": cfg.ApprovalMode,
	})

	// Connect to MCP servers.
	registry := mcp.NewRegistry(log)
	defer registry.Shutdown()

	if err := registry.LoadConfig(cfg.MCPConfigPath); err != nil {
		log.Error("failed to load MCP config", map[string]string{"error": err.Error()})
		fmt.Fprintf(os.Stderr, "Error loading MCP config: %v\n", err)
		os.Exit(1)
	}

	tools := registry.ListTools()
	if len(tools) == 0 {
		log.Warn("no tools discovered from MCP servers")
		fmt.Fprintln(os.Stderr, "Warning: no tools discovered. Check MCP server configuration.")
	}

	// --check mode: just list tools and exit.
	if *checkFlag {
		fmt.Printf("\n✅ Connected MCP servers. Discovered %d tools:\n\n", len(tools))
		for _, t := range tools {
			fmt.Printf("  • %-40s %s\n", t.Name, t.Description)
		}
		fmt.Println()
		return
	}

	// Get task from arguments.
	task := strings.Join(flag.Args(), " ")
	if task == "" {
		fmt.Fprintln(os.Stderr, "Usage: go-mcp [--check] \"<task description>\"")
		fmt.Fprintln(os.Stderr, "Example: go-mcp \"open the browser and search for Go MCP libraries\"")
		os.Exit(1)
	}

	// Set up components.
	llmClient := llm.NewClient(
		cfg.OpenAIBaseURL,
		cfg.OpenAIAPIKey,
		cfg.OpenAIModel,
		cfg.RequestTimeoutSeconds,
		log,
	)

	ctxMgr := agentctx.New(systemPrompt, 100) // Keep up to 100 messages.
	permMgr := permission.NewManager(cfg.ApprovalMode, cfg.AllowShell, log)
	screenshotH := screenshot.NewHandler(registry, log)

	loop := agent.NewLoop(
		llmClient,
		registry,
		ctxMgr,
		permMgr,
		screenshotH,
		log,
		cfg.MaxSteps,
		cfg.MaxToolCalls,
	)

	// Run with graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nShutting down...")
		cancel()
	}()

	fmt.Printf("🤖 Task: %s\n\n", task)
	result, err := loop.RunTask(ctx, task)
	if err != nil {
		log.Error("task failed", map[string]string{"error": err.Error()})
		fmt.Fprintf(os.Stderr, "\n❌ Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Result:\n%s\n", result)
}
