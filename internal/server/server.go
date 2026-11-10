package server

import (
  "context"
  "fmt"
  "net/http"
  "sync"
  "os"
  "path/filepath"
  "strconv"

  "github.com/Mohamed-faaris/go-mcp/internal/coordination"
  "github.com/Mohamed-faaris/go-mcp/internal/credentials"
  "github.com/Mohamed-faaris/go-mcp/internal/llm"
  "github.com/Mohamed-faaris/go-mcp/internal/logger"
  "github.com/Mohamed-faaris/go-mcp/internal/podman"
  "github.com/Mohamed-faaris/go-mcp/internal/task"
  "github.com/Mohamed-faaris/go-mcp/internal/tools"
  "github.com/Mohamed-faaris/go-mcp/internal/webconfig"
  "github.com/Mohamed-faaris/go-mcp/internal/worker"
)

type Server struct {
  cfg *webconfig.Config
  podman *podman.Client
  log *logger.Logger
  registry *mcpRegistry
  mux *http.ServeMux
  tasks *task.Store
  coordination *coordination.Store
  toolPolicies *tools.Store
  agentRunner *worker.Runner
  githubApp *credentials.GitHubAppBroker
  taskCancelMu sync.Mutex
  taskCancels map[string]context.CancelFunc
}

func dataPath(name string)string{
  root:=os.Getenv("GOMCP_DATA_DIR");if root==""{root="data"}
  return filepath.Join(root,name)
}

func New(cfg *webconfig.Config,pc *podman.Client,log *logger.Logger)*Server{
  s:=&Server{cfg:cfg,podman:pc,log:log,registry:loadMCPRegistry(cfg),mux:http.NewServeMux(),taskCancels:map[string]context.CancelFunc{}}
  if store,err:=task.OpenStore(dataPath("tasks.json"));err!=nil{log.Error("task store initialization failed",map[string]string{"error":err.Error()})}else{s.tasks=store}
  if store,err:=coordination.OpenStore(dataPath("coordination.json"));err!=nil{log.Error("agent coordination store initialization failed",map[string]string{"error":err.Error()})}else{s.coordination=store}
  if store,err:=tools.OpenStore(dataPath("tool_policies.json"));err!=nil{log.Error("tool policy store initialization failed",map[string]string{"error":err.Error()})}else{s.toolPolicies=store}
  base,key,model:=os.Getenv("OPENAI_BASE_URL"),os.Getenv("OPENAI_API_KEY"),os.Getenv("OPENAI_MODEL")
  if base!=""&&key!=""&&model!=""{
    timeout:=90;if v,e:=strconv.Atoi(os.Getenv("REQUEST_TIMEOUT_SECONDS"));e==nil&&v>0{timeout=v}
    s.agentRunner=worker.New(llm.NewClient(base,key,model,timeout,log),pc,log);s.agentRunner.SetToolPolicies(s.toolPolicies)
  }
  if broker,err:=credentials.NewGitHubAppBrokerFromEnv();err==nil{s.githubApp=broker}
  s.registerRoutes()
  return s
}

func(s *Server)registerRoutes(){
  s.mux.HandleFunc("POST /api/auth/login",s.handleLogin)
  s.mux.HandleFunc("GET /api/auth/me",s.withAuth(s.handleMe,"viewer"))

  s.mux.HandleFunc("GET /api/containers",s.withAuth(s.handleListContainers,"viewer"))
  s.mux.HandleFunc("GET /api/containers/{id}",s.withAuth(s.handleGetContainer,"viewer"))
  s.mux.HandleFunc("GET /api/containers/{id}/logs",s.withAuth(s.handleContainerLogs,"viewer"))
  s.mux.HandleFunc("GET /api/containers/{id}/terminal",s.withAuth(s.handleTerminal,"viewer"))
  s.mux.HandleFunc("POST /api/containers",s.withAuth(s.handleCreateContainer,"admin"))
  s.mux.HandleFunc("POST /api/containers/{id}/start",s.withAuth(s.handleStartContainer,"admin"))
  s.mux.HandleFunc("POST /api/containers/{id}/stop",s.withAuth(s.handleStopContainer,"admin"))
  s.mux.HandleFunc("POST /api/containers/{id}/duplicate",s.withAuth(s.handleDuplicateContainer,"admin"))
  s.mux.HandleFunc("DELETE /api/containers/{id}",s.withAuth(s.handleDeleteContainer,"admin"))

  s.mux.HandleFunc("GET /api/projects",s.withAuth(s.handleListProjects,"viewer"))
  s.mux.HandleFunc("GET /api/projects/browse",s.withAuth(s.handleBrowseProjects,"viewer"))
  s.mux.HandleFunc("POST /api/projects/clone",s.withAuth(s.handleCloneProject,"admin"))
  s.mux.HandleFunc("GET /api/mcp-tools",s.withAuth(s.handleListMCPTools,"viewer"))
  s.mux.HandleFunc("POST /api/mcp-tools",s.withAuth(s.handleCreateMCPTool,"viewer"))
  s.mux.HandleFunc("PUT /api/mcp-tools/{id}",s.withAuth(s.handleUpdateMCPTool,"viewer"))
  s.mux.HandleFunc("DELETE /api/mcp-tools/{id}",s.withAuth(s.handleDeleteMCPTool,"viewer"))
  s.mux.HandleFunc("GET /api/images",s.withAuth(s.handleListImages,"viewer"))
  s.mux.HandleFunc("POST /api/images/pull",s.withAuth(s.handlePullImage,"admin"))
  s.mux.HandleFunc("GET /api/devices",s.withAuth(s.handleListDevices,"viewer"))
  s.mux.HandleFunc("GET /api/config",s.withAuth(s.handleGetConfig,"viewer"))

  // Versioned autonomous task API.
  s.mux.HandleFunc("GET /api/v1/presets",s.withAuth(s.handleListPresets,"viewer"))
  s.mux.HandleFunc("GET /api/v1/tasks",s.withAuth(s.handleListTasks,"viewer"))
  s.mux.HandleFunc("POST /api/v1/tasks",s.withAuth(s.handleCreateTask,"viewer"))
  s.mux.HandleFunc("GET /api/v1/tasks/{id}",s.withAuth(s.handleGetTask,"viewer"))
  s.mux.HandleFunc("GET /api/v1/tasks/{id}/report",s.withAuth(s.handleGetTaskReport,"viewer"))
  s.mux.HandleFunc("POST /api/v1/tasks/{id}/cancel",s.withAuth(s.handleCancelTask,"viewer"))
  s.mux.HandleFunc("GET /api/v1/projects/git",s.withAuth(s.handleInspectProjectGit,"viewer"))

  // Native tool catalog and approval inbox.
  s.mux.HandleFunc("GET /api/v1/tools",s.withAuth(s.handleListToolPolicies,"viewer"))
  s.mux.HandleFunc("PATCH /api/v1/tools/{id}",s.withAuth(s.handleUpdateToolPolicy,"admin"))
  s.mux.HandleFunc("GET /api/v1/approvals",s.withAuth(s.handleListApprovals,"viewer"))
  s.mux.HandleFunc("POST /api/v1/approvals/request",s.withAuth(s.handleRequestApproval,"viewer"))
  s.mux.HandleFunc("POST /api/v1/approvals/{id}/approve",s.withAuth(s.handleApprove,"admin"))
  s.mux.HandleFunc("POST /api/v1/approvals/{id}/deny",s.withAuth(s.handleDeny,"admin"))

  // Multi-agent sessions, messaging, leases, and durable handoffs.
  s.mux.HandleFunc("POST /api/v1/agent-sessions",s.withAuth(s.handleRegisterAgent,"viewer"))
  s.mux.HandleFunc("POST /api/v1/agent-sessions/{id}/heartbeat",s.withAuth(s.handleAgentHeartbeat,"viewer"))
  s.mux.HandleFunc("POST /api/v1/agent-sessions/{id}/resume",s.withAuth(s.handleResumeAgent,"viewer"))
  s.mux.HandleFunc("GET /api/v1/agent-sessions/{id}/context",s.withAuth(s.handleAgentContext,"viewer"))
  s.mux.HandleFunc("POST /api/v1/agent-sessions/{id}/handoffs",s.withAuth(s.handleCreateHandoff,"viewer"))
  s.mux.HandleFunc("GET /api/v1/projects/{projectID}/agents",s.withAuth(s.handleListProjectAgents,"viewer"))
  s.mux.HandleFunc("GET /api/v1/projects/{projectID}/context",s.withAuth(s.handleProjectContext,"viewer"))
  s.mux.HandleFunc("GET /api/v1/projects/{projectID}/messages",s.withAuth(s.handleReadAgentMessages,"viewer"))
  s.mux.HandleFunc("POST /api/v1/projects/{projectID}/messages",s.withAuth(s.handleSendAgentMessage,"viewer"))
  s.mux.HandleFunc("POST /api/v1/work-items/claim",s.withAuth(s.handleClaimWorkItem,"viewer"))
  s.mux.HandleFunc("DELETE /api/v1/work-items/claims/{id}",s.withAuth(s.handleReleaseWorkItem,"viewer"))

  s.mux.Handle("/",s.spaHandler("web/dist"))
}

func(s *Server)Start()error{
  addr:=fmt.Sprintf("%s:%d",s.cfg.Server.Host,s.cfg.Server.Port)
  s.log.Info("web server listening",map[string]interface{}{"addr":addr,"port":s.cfg.Server.Port})
  return http.ListenAndServe(addr,s.mux)
}
