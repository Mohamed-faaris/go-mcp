export interface AuthUser {
  username: string;
  role: 'admin' | 'viewer';
}

export interface Mount {
  Type: string;
  Source: string;
  Destination: string;
  Mode?: string;
  RW?: boolean;
}

export interface ContainerSummary {
  Id: string;
  Names: string[];
  Image: string;
  State: string;
  Status: string;
  Created: number;
  Labels?: Record<string, string>;
  Mounts?: string[];
  projectDir?: string;
}

export interface ContainerState {
  Status: string;
  Running: boolean;
  ExitCode: number;
  StartedAt: string;
  FinishedAt: string;
}

export interface ContainerInspect {
  Id: string;
  Name: string;
  Created: string;
  State: ContainerState;
  Config: {
    Image: string;
    Cmd: string[];
    Env: string[];
    Labels: Record<string, string>;
  };
  HostConfig: {
    Memory: number;
    NanoCpus: number;
  };
  Mounts: Mount[];
}

export interface MCPToolSpec {
  name: string;
  command: string;
  args: string[];
}

export interface MCPRegistryTool {
  id: string;
  name: string;
  command: string;
  args: string[];
  level: 'admin' | 'user';
  description: string;
  builtIn: boolean;
  createdBy: string;
  createdAt: string;
}

export interface MCPToolsResponse {
  tools: MCPRegistryTool[];
}

export interface CreateContainerRequest {
  name: string;
  image: string;
  projectDir: string;
  memoryLimit: string;
  cpuLimit: string;
  autoStart: boolean;
  mcpTools: MCPToolSpec[];
  devices: string[];
}

export interface ProjectItem {
  name: string;
  path: string;
  type: 'project' | 'tmp';
  modified: string;
}

export interface ProjectsResponse {
  projectsDir: string;
  tmpDir: string;
  items: ProjectItem[];
}

export interface ConfiguredImage {
  name: string;
  image: string;
}

export interface LocalImage {
  Id: string;
  RepoTags?: string[];
  Size: number;
  Created: number;
}

export interface ImagesResponse {
  configured: ConfiguredImage[];
  local: LocalImage[];
}

export interface HostDevice {
  name: string;
  path: string;
  type: 'video' | 'media';
  busInfo?: string;
}

export interface PublicConfig {
  defaults: {
    base_image: string;
    resource_limits: {
      memory: string;
      cpus: string;
    };
    mcp_tools: MCPToolSpec[];
  };
  directories: {
    projects: string;
    tmp_podmans: string;
  };
  images: ConfiguredImage[];
  authMode: string;
}


export interface RuntimePreset {
  id: string;
  name: string;
  description: string;
  image: string;
  setupCommand: string;
  defaultMemory: string;
  defaultCpus: string;
  packageManager: string;
}

export interface TaskItem {
  id: string;
  ownerId: string;
  projectId: string;
  projectPath: string;
  prompt: string;
  presetId: string;
  status: string;
  containerId?: string;
  workspacePath?: string;
  remoteUrl?: string;
  branchName?: string;
  baseCommit?: string;
  resultCommit?: string;
  summary?: string;
  error?: string;
  keepContainer?: boolean;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
  updatedAt: string;
}

export interface TaskCheck {
  name: string;
  command?: string;
  status: string;
  exitCode?: number;
  duration?: string;
  output?: string;
}

export interface TaskReport {
  taskId: string;
  summary: string;
  changedFiles: string[];
  checks: TaskCheck[];
  commits: string[];
  warnings: string[];
  artifacts: string[];
  startedAt: string;
  finishedAt: string;
}

export interface ToolDefinition {
  id: string;
  name: string;
  description: string;
  category: string;
  scope: string;
  risk: string;
  source: string;
  enabled: boolean;
  policy: 'automatic'|'approval_required'|'disabled'|'hard_denied';
  hardDenied: boolean;
  updatedAt: string;
}

export interface ApprovalItem {
  id: string;
  taskId: string;
  agentSessionId?: string;
  toolId: string;
  resourceType: string;
  resourceId: string;
  argumentsDigest: string;
  summary: string;
  status: string;
  requestedBy: string;
  reviewedBy?: string;
  createdAt: string;
  expiresAt: string;
  reviewedAt?: string;
}

export interface AgentSession {
  id: string;
  ownerId: string;
  projectId: string;
  taskId?: string;
  workstreamId?: string;
  agentLabel: string;
  branch?: string;
  commit?: string;
  status: string;
  lastHeartbeatAt: string;
  createdAt: string;
}

export interface AgentMessage {
  id: string;
  cursor: number;
  projectId: string;
  taskId?: string;
  channel: string;
  senderSessionId: string;
  recipientSessionId?: string;
  type: string;
  body: string;
  correlationId?: string;
  createdAt: string;
}
