package worker

import (
  "context"
  "encoding/base64"
  "encoding/json"
  "errors"
  "fmt"
  "path"
  "strings"
  "time"

  "github.com/Mohamed-faaris/go-mcp/internal/llm"
  "github.com/Mohamed-faaris/go-mcp/internal/logger"
  "github.com/Mohamed-faaris/go-mcp/internal/podman"
  "github.com/Mohamed-faaris/go-mcp/internal/task"
)

type Runner struct {
  llm *llm.Client
  podman *podman.Client
  log *logger.Logger
  maxSteps int
  maxCalls int
}
type Result struct { Summary string; Checks []task.CheckResult; Warnings []string }

func New(llmClient *llm.Client, pc *podman.Client, log *logger.Logger) *Runner {
  return &Runner{llm:llmClient,podman:pc,log:log,maxSteps:30,maxCalls:80}
}

func tool(name,description string,properties map[string]interface{},required ...string)llm.ToolDef{
  return llm.ToolDef{Type:"function",Function:llm.FunctionInfo{Name:name,Description:description,Parameters:map[string]interface{}{"type":"object","properties":properties,"required":required}}}
}
func defs()[]llm.ToolDef{
  stringProp:=func(desc string)map[string]interface{}{return map[string]interface{}{"type":"string","description":desc}}
  return []llm.ToolDef{
    tool("run_command","Run a command inside this task's isolated container. Working directory is /workspace. The command cannot access the host.",map[string]interface{}{"command":stringProp("Shell command to run in the task container"),"timeout_seconds":map[string]interface{}{"type":"integer","description":"Maximum runtime, capped at 300 seconds"}},"command"),
    tool("read_file","Read a file inside /workspace. Paths must be relative to the workspace.",map[string]interface{}{"path":stringProp("Relative file path")},"path"),
    tool("write_file","Write a text file inside /workspace. Paths must be relative to the workspace.",map[string]interface{}{"path":stringProp("Relative file path"),"content":stringProp("Complete UTF-8 file content")},"path","content"),
    tool("list_files","List files below a workspace-relative directory.",map[string]interface{}{"path":stringProp("Relative directory path, default .")}),
    tool("search_files","Search text inside workspace files while skipping dependency and build directories.",map[string]interface{}{"query":stringProp("Text or regular expression to search for"),"path":stringProp("Relative search root, default .")},"query"),
    tool("git_status","Show Git status and current branch inside the project workspace.",map[string]interface{}{}),
    tool("git_diff","Show the current working tree diff. Use this to review changes before finishing.",map[string]interface{}{"staged":map[string]interface{}{"type":"boolean","description":"Show staged diff instead of unstaged diff"}}),
  }
}
func validPath(p string)bool{
  p=strings.TrimSpace(p);if p==""||strings.ContainsRune(p,0)||strings.HasPrefix(p,"/")||strings.Contains(p,"\\"){return false}
  clean:=path.Clean(p);return clean!=".."&&!strings.HasPrefix(clean,"../")
}
func shellQuote(s string)string{return "'"+strings.ReplaceAll(s,"'","'\\''")+"'"}
func argString(raw string,key string)(string,error){var a map[string]interface{};if e:=json.Unmarshal([]byte(raw),&a);e!=nil{return "",e};v,ok:=a[key].(string);if !ok{return "",fmt.Errorf("%s must be a string",key)};return v,nil}
func (r *Runner) execute(ctx context.Context,containerID,name,args string)(string,int,error){
  switch name{
  case "run_command":
    command,e:=argString(args,"command");if e!=nil{return "",1,e};if strings.TrimSpace(command)==""||len(command)>20000{return "",1,errors.New("command is empty or exceeds 20,000 characters")}
    timeout:=120*time.Second;var a map[string]interface{};_=json.Unmarshal([]byte(args),&a);if v,ok:=a["timeout_seconds"].(float64);ok{seconds:=int(v);if seconds>0{if seconds>300{seconds=300};timeout=time.Duration(seconds)*time.Second}}
    res,e:=r.podman.RunCommand(ctx,containerID,command,timeout);return strings.TrimSpace(res.Stdout+"\n"+res.Stderr),res.ExitCode,e
  case "read_file":
    p,e:=argString(args,"path");if e!=nil||!validPath(p){return "",1,errors.New("invalid workspace path")};res,e:=r.podman.RunCommand(ctx,containerID,"cat -- "+shellQuote(p),30*time.Second);return res.Stdout+res.Stderr,res.ExitCode,e
  case "write_file":
    p,e:=argString(args,"path");if e!=nil||!validPath(p){return "",1,errors.New("invalid workspace path")};content,e:=argString(args,"content");if e!=nil{return "",1,e};encoded:=base64.StdEncoding.EncodeToString([]byte(content));cmd:="mkdir -p -- "+shellQuote(path.Dir(p))+" && printf %s "+shellQuote(encoded)+" | base64 -d > "+shellQuote(p);res,e:=r.podman.RunCommand(ctx,containerID,cmd,30*time.Second);return strings.TrimSpace(res.Stdout+res.Stderr),res.ExitCode,e
  case "list_files":
    p:=".";var a map[string]interface{};_=json.Unmarshal([]byte(args),&a);if v,ok:=a["path"].(string);ok&&v!=""{p=v};if !validPath(p){return "",1,errors.New("invalid workspace path")};res,e:=r.podman.RunCommand(ctx,containerID,"find "+shellQuote(p)+" -maxdepth 4 -type f -not -path '*/node_modules/*' -not -path '*/.git/*' | head -n 400",30*time.Second);return res.Stdout+res.Stderr,res.ExitCode,e
  case "search_files":
    q,e:=argString(args,"query");if e!=nil{return "",1,e};if len(q)>2000{return "",1,errors.New("query too long")};p:=".";var a map[string]interface{};_=json.Unmarshal([]byte(args),&a);if v,ok:=a["path"].(string);ok&&v!=""{p=v};if !validPath(p){return "",1,errors.New("invalid workspace path")};res,e:=r.podman.RunCommand(ctx,containerID,"grep -RIn --exclude-dir=.git --exclude-dir=node_modules --exclude-dir=dist --exclude-dir=build -- "+shellQuote(q)+" "+shellQuote(p)+" 2>/dev/null | head -n 300",30*time.Second);return res.Stdout+res.Stderr,res.ExitCode,e
  case "git_status":
    res,e:=r.podman.RunCommand(ctx,containerID,"git status --short --branch",30*time.Second);return res.Stdout+res.Stderr,res.ExitCode,e
  case "git_diff":
    staged:=false;var a map[string]interface{};_=json.Unmarshal([]byte(args),&a);if v,ok:=a["staged"].(bool);ok{staged=v};cmd:="git diff --";if staged{cmd="git diff --cached --"};res,e:=r.podman.RunCommand(ctx,containerID,cmd+" | head -c 100000",30*time.Second);return res.Stdout+res.Stderr,res.ExitCode,e
  default:return "",1,fmt.Errorf("unknown tool %q",name)
  }
}

func (r *Runner) Run(ctx context.Context,t *task.Task)(Result,error){
  if r.llm==nil{return Result{},errors.New("LLM provider is not configured (OPENAI_BASE_URL, OPENAI_API_KEY, OPENAI_MODEL)")}
  system:="You are a careful autonomous software engineer working inside one isolated project container. Your entire filesystem and terminal scope is /workspace. You cannot access the host or other containers.\n\n"+
    "Work method:\n1. Inspect the repository, project instructions, package metadata, and existing changes.\n2. Create a concise plan, then implement the user's task.\n3. Preserve existing conventions and avoid unrelated changes.\n4. Run relevant formatting, lint, type checks, builds, and tests. Report commands and observed outcomes accurately.\n5. Inspect git diff and status before finishing. Do not push to a remote or access the host, Podman socket, credentials, or other tasks.\n6. Finish with a concise summary, changed paths, commands/checks, and remaining risks.\n\nUse tools for all filesystem and command operations. The workspace is already checked out on the task branch. Do not assume a command succeeded; inspect its returned exit code and output."
  messages:=[]llm.Message{{Role:"system",Content:system},{Role:"user",Content:"Task: "+t.Prompt+"\nProject: "+t.ProjectPath+"\nPreset: "+t.PresetID+"\nBranch: "+t.BranchName}}
  definitions:=defs();calls:=0;var final string;checks:=[]task.CheckResult{}
  for step:=0;step<r.maxSteps;step++{
    choice,_,err:=r.llm.Complete(ctx,messages,definitions);if err!=nil{return Result{},fmt.Errorf("LLM request: %w",err)}
    messages=append(messages,choice.Message)
    if len(choice.Message.ToolCalls)==0{
      if s,ok:=choice.Message.Content.(string);ok{final=s}else{final=fmt.Sprint(choice.Message.Content)}
      if choice.FinishReason=="stop"||choice.FinishReason=="end_turn"{break}
      if step==r.maxSteps-1{return Result{},errors.New("agent exceeded maximum planning steps")}
    }
    for _,call:=range choice.Message.ToolCalls{
      calls++;if calls>r.maxCalls{return Result{},errors.New("agent exceeded maximum tool calls")}
      start:=time.Now();output,exitCode,toolErr:=r.execute(ctx,t.ContainerID,call.Function.Name,call.Function.Arguments);duration:=time.Since(start)
      status:="passed";if exitCode!=0||toolErr!=nil{status="failed"}
      if call.Function.Name=="run_command"{cmd,_:=argString(call.Function.Arguments,"command");lower:=strings.ToLower(cmd);if strings.Contains(lower,"test")||strings.Contains(lower,"build")||strings.Contains(lower,"lint")||strings.Contains(lower,"check"){code:=exitCode;checks=append(checks,task.CheckResult{Name:"command",Command:cmd,Status:status,ExitCode:&code,Duration:duration.String(),Output:truncate(output,8000)})}}
      if toolErr!=nil{output+="\nTool execution error: "+toolErr.Error()}
      output+="\nExit code: "+fmt.Sprint(exitCode)
      messages=append(messages,llm.Message{Role:"tool",ToolCallID:call.ID,Content:output})
    }
  }
  if final==""{return Result{},errors.New("agent returned no final response")}
  return Result{Summary:final,Checks:checks},nil
}
func truncate(s string,n int)string{if len(s)<=n{return s};return s[:n]+"\n...[truncated]"}
