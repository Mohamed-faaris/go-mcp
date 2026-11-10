package server

import (
  "context"
  "crypto/sha256"
  "encoding/hex"
  "encoding/json"
  "fmt"
  "net/http"
  "os"
  "os/exec"
  "path/filepath"
  "strconv"
  "strings"
  "time"

  "github.com/Mohamed-faaris/go-mcp/internal/coordination"
  gitops "github.com/Mohamed-faaris/go-mcp/internal/git"
  "github.com/Mohamed-faaris/go-mcp/internal/presets"
  "github.com/Mohamed-faaris/go-mcp/internal/task"
  "github.com/Mohamed-faaris/go-mcp/internal/tools"
  "github.com/Mohamed-faaris/go-mcp/internal/workspace"
)

func(s *Server) handleListPresets(w http.ResponseWriter,r *http.Request){s.jsonOK(w,map[string]interface{}{"presets":presets.List()})}

func withinRoot(root,candidate string)bool{
  if root==""||candidate==""{return false}
  rel,err:=filepath.Rel(root,candidate);if err!=nil{return false}
  return rel=="."||(rel!=".."&&!strings.HasPrefix(rel,".."+string(filepath.Separator))&&!filepath.IsAbs(rel))
}

func(s *Server) resolveProjectPath(raw string)(string,error){
  if strings.TrimSpace(raw)==""{return "",fmt.Errorf("projectPath is required")}
  abs,err:=filepath.Abs(raw);if err!=nil{return "",err}
  resolved,err:=filepath.EvalSymlinks(abs);if err!=nil{return "",fmt.Errorf("cannot resolve project path: %w",err)}
  info,err:=os.Stat(resolved);if err!=nil||!info.IsDir(){return "",fmt.Errorf("project path must be an existing directory")}
  roots:=[]string{s.cfg.Directories.Projects,s.cfg.Directories.TmpPodmans}
  for _,root:=range roots{
    if root==""{continue};rootAbs,e:=filepath.Abs(root);if e!=nil{continue};rootResolved,e:=filepath.EvalSymlinks(rootAbs);if e!=nil{rootResolved=rootAbs}
    if withinRoot(rootResolved,resolved){return resolved,nil}
  }
  return "",fmt.Errorf("project path must be inside configured projects or scratch directories")
}

func projectIDFor(path string)string{sum:=sha256.Sum256([]byte(filepath.Clean(path)));return hex.EncodeToString(sum[:12])}

func(s *Server) handleInspectProjectGit(w http.ResponseWriter,r *http.Request){
  p,err:=s.resolveProjectPath(r.URL.Query().Get("path"));if err!=nil{s.jsonError(w,err.Error(),http.StatusBadRequest);return}
  remotes,err:=gitops.ReadRemotes(r.Context(),p);if err!=nil{s.jsonError(w,err.Error(),http.StatusInternalServerError);return}
  preset,detected:=presets.Detect(p)
  branch,_:=gitops.Branch(r.Context(),p);commit,_:=gitops.HeadCommit(r.Context(),p)
  s.jsonOK(w,map[string]interface{}{"isGit":gitops.IsGitRepository(p),"remotes":remotes,"branch":branch,"commit":commit,"detectedPreset":preset,"presetDetected":detected})
}

func(s *Server) handleListTasks(w http.ResponseWriter,r *http.Request){
  if s.tasks==nil{s.jsonError(w,"task store unavailable",http.StatusServiceUnavailable);return}
  owner:="";if getUserRole(r.Context())!="admin"{owner=getUserName(r.Context())}
  s.jsonOK(w,map[string]interface{}{"tasks":s.tasks.List(owner)})
}

func(s *Server) handleCreateTask(w http.ResponseWriter,r *http.Request){
  if s.tasks==nil{s.jsonError(w,"task store unavailable",http.StatusServiceUnavailable);return}
  var req struct{
    ProjectPath string `json:"projectPath"`
    Prompt string `json:"prompt"`
    PresetID string `json:"presetId"`
    KeepContainer bool `json:"keepContainer"`
  }
  if err:=json.NewDecoder(http.MaxBytesReader(w,r.Body,128*1024)).Decode(&req);err!=nil{s.jsonError(w,"invalid request body",http.StatusBadRequest);return}
  req.Prompt=strings.TrimSpace(req.Prompt)
  if req.Prompt==""||len(req.Prompt)>12000{s.jsonError(w,"prompt is required and must be at most 12,000 characters",http.StatusBadRequest);return}
  projectPath,err:=s.resolveProjectPath(req.ProjectPath);if err!=nil{s.jsonError(w,err.Error(),http.StatusBadRequest);return}
  if req.PresetID==""{detected,_:=presets.Detect(projectPath);req.PresetID=detected.ID}
  preset,ok:=presets.Get(req.PresetID);if !ok{s.jsonError(w,"unknown runtime preset",http.StatusBadRequest);return}
  now:=time.Now().UTC();item,err:=task.New(task.NewID(),getUserName(r.Context()),projectIDFor(projectPath),req.Prompt,now)
  if err!=nil{s.jsonError(w,err.Error(),http.StatusBadRequest);return}
  item.ProjectPath=projectPath;item.PresetID=preset.ID;item.KeepContainer=req.KeepContainer
  if err=s.tasks.Create(item);err!=nil{s.jsonError(w,fmt.Sprintf("cannot persist task: %v",err),http.StatusInternalServerError);return}
  go s.runTask(item.ID)
  w.WriteHeader(http.StatusAccepted)
  s.jsonOK(w,map[string]interface{}{"task":item})
}

func(s *Server) taskForRequest(w http.ResponseWriter,r *http.Request)(*task.Task,bool){
  if s.tasks==nil{s.jsonError(w,"task store unavailable",http.StatusServiceUnavailable);return nil,false}
  item,ok:=s.tasks.Get(r.PathValue("id"));if !ok{s.jsonError(w,"task not found",http.StatusNotFound);return nil,false}
  if getUserRole(r.Context())!="admin"&&item.OwnerID!=getUserName(r.Context()){s.jsonError(w,"task not found",http.StatusNotFound);return nil,false}
  return item,true
}
func(s *Server) handleGetTask(w http.ResponseWriter,r *http.Request){item,ok:=s.taskForRequest(w,r);if !ok{return};s.jsonOK(w,map[string]interface{}{"task":item})}
func(s *Server) handleGetTaskReport(w http.ResponseWriter,r *http.Request){
  item,ok:=s.taskForRequest(w,r);if !ok{return};report,found:=s.tasks.GetReport(item.ID);if !found{s.jsonError(w,"task report is not available yet",http.StatusNotFound);return};s.jsonOK(w,report)
}

func(s *Server) handleCancelTask(w http.ResponseWriter,r *http.Request){
  item,ok:=s.taskForRequest(w,r);if !ok{return}
  s.taskCancelMu.Lock();cancel:=s.taskCancels[item.ID];s.taskCancelMu.Unlock()
  if cancel!=nil{cancel()}
  updated,err:=s.tasks.Update(item.ID,func(t *task.Task)error{
    if t.Status==task.StatusCompleted||t.Status==task.StatusFailed||t.Status==task.StatusCancelled{return fmt.Errorf("task is already %s",t.Status)}
    return t.Transition(task.StatusCancelled,time.Now().UTC())
  })
  if err!=nil{s.jsonError(w,err.Error(),http.StatusConflict);return}
  s.jsonOK(w,map[string]interface{}{"task":updated})
}

func(s *Server) setTaskFailed(id string,runErr error){
  _,_=s.tasks.Update(id,func(t *task.Task)error{
    if t.Status==task.StatusCompleted||t.Status==task.StatusCancelled||t.Status==task.StatusFailed{return nil}
    t.Error=runErr.Error()
    if task.CanTransition(t.Status,task.StatusFailed){return t.Transition(task.StatusFailed,time.Now().UTC())}
    return nil
  })
}
func(s *Server) runTask(id string){
  if s.tasks==nil{return}
  ctx,cancel:=context.WithTimeout(context.Background(),45*time.Minute)
  s.taskCancelMu.Lock();s.taskCancels[id]=cancel;s.taskCancelMu.Unlock()
  defer func(){cancel();s.taskCancelMu.Lock();delete(s.taskCancels,id);s.taskCancelMu.Unlock()}()
  item,ok:=s.tasks.Get(id);if !ok{return}
  if item.Status==task.StatusCancelled{return}
  _,err:=s.tasks.Update(id,func(t *task.Task)error{return t.Transition(task.StatusProvisioning,time.Now().UTC())})
  if err!=nil{return}
  if s.agentRunner==nil{s.setTaskFailed(id,fmt.Errorf("LLM provider is not configured; set OPENAI_BASE_URL, OPENAI_API_KEY and OPENAI_MODEL"));return}
  preset,ok:=presets.Get(item.PresetID);if !ok{s.setTaskFailed(id,fmt.Errorf("runtime preset %q no longer exists",item.PresetID));return}
  workspaceRoot:=filepath.Join(s.cfg.Directories.TmpPodmans,"go-mcp-workspaces")
  prepared,err:=workspace.Prepare(ctx,item.ProjectPath,workspaceRoot,id);if err!=nil{s.setTaskFailed(id,fmt.Errorf("prepare workspace: %w",err));return}
  _,err=s.tasks.Update(id,func(t *task.Task)error{t.WorkspacePath=prepared.Path;t.RemoteURL=prepared.Remote;t.BranchName=prepared.Branch;t.BaseCommit=prepared.BaseCommit;return nil})
  if err!=nil{s.setTaskFailed(id,err);return}
  labels:=map[string]string{
    "gomcp.managed":"true","gomcp.owner_id":item.OwnerID,"gomcp.task_id":item.ID,
    "gomcp.project_id":item.ProjectID,"gomcp.preset_id":preset.ID,
    "gomcp.created_at":time.Now().UTC().Format(time.RFC3339),
  }
  name:="task-"+id;if len(name)>32{name=name[:32]}
  containerID,err:=s.podman.CreateContainer(ctx,podman.CreateContainerOpts{
    Name:name,Image:preset.Image,ProjectDir:prepared.Path,MemoryLimit:preset.DefaultMemory,CPULimit:preset.DefaultCPUs,Labels:labels,
  })
  if err!=nil&&strings.Contains(strings.ToLower(err.Error()),"no such image"){
    if pullErr:=s.podman.PullImage(ctx,preset.Image);pullErr==nil{
      containerID,err=s.podman.CreateContainer(ctx,podman.CreateContainerOpts{Name:name,Image:preset.Image,ProjectDir:prepared.Path,MemoryLimit:preset.DefaultMemory,CPULimit:preset.DefaultCPUs,Labels:labels})
    }
  }
  if err!=nil{s.setTaskFailed(id,fmt.Errorf("create task container: %w",err));return}
  _,_=s.tasks.Update(id,func(t *task.Task)error{t.ContainerID=containerID;return nil})
  if err=s.podman.StartContainer(ctx,containerID);err!=nil{s.setTaskFailed(id,fmt.Errorf("start task container: %w",err));_=s.podman.RemoveContainer(context.Background(),containerID);return}
  _,_=s.tasks.Update(id,func(t *task.Task)error{return t.Transition(task.StatusRunning,time.Now().UTC())})
  warnings:=[]string{}
  setupCtx,setupCancel:=context.WithTimeout(ctx,8*time.Minute)
  setup,setupErr:=s.podman.RunCommand(setupCtx,containerID,preset.SetupCommand,8*time.Minute);setupCancel()
  if setupErr!=nil||setup.ExitCode!=0{warnings=append(warnings,"Preset setup command did not complete successfully: "+truncate(setup.Stderr+setup.Stdout,2000))}
  result,runErr:=s.agentRunner.Run(ctx,itemWithContainer(item,containerID,prepared.Branch,preset.ID))
  if ctx.Err()!=nil{
    current,_:=s.tasks.Get(id);if current!=nil&&current.Status!=task.StatusCancelled{s.setTaskFailed(id,ctx.Err())}
    return
  }
  if runErr!=nil{s.setTaskFailed(id,runErr);if !item.KeepContainer{_=s.podman.StopContainer(context.Background(),containerID);_=s.podman.RemoveContainer(context.Background(),containerID)};return}
  _,_=s.tasks.Update(id,func(t *task.Task)error{return t.Transition(task.StatusValidating,time.Now().UTC())})
  commit,warn,changed:=finalizeGit(ctx,prepared.Path,prepared.BaseCommit,id)
  warnings=append(warnings,warn...)
  report:=task.Report{TaskID:id,Summary:result.Summary,ChangedFiles:changed,Checks:result.Checks,Warnings:warnings,Artifacts:[]string{},StartedAt:item.CreatedAt,FinishedAt:time.Now().UTC()}
  if commit!=""{report.Commits=[]string{commit}}
  if len(report.Checks)==0{report.Checks=[]task.CheckResult{{Name:"Automated validation",Status:"not_run",Output:"The agent did not execute a command identified as a test, build, lint or check."}}}
  _=s.tasks.SaveReport(report)
  _,_=s.tasks.Update(id,func(t *task.Task)error{t.ResultCommit=commit;t.Summary=result.Summary;return t.Transition(task.StatusCompleted,time.Now().UTC())})
  if !item.KeepContainer{
    stopCtx,stopCancel:=context.WithTimeout(context.Background(),20*time.Second);_=s.podman.StopContainer(stopCtx,containerID);_=s.podman.RemoveContainer(stopCtx,containerID);stopCancel()
  }
}

func itemWithContainer(item *task.Task,containerID,branch,presetID string)*task.Task{copy:=*item;copy.ContainerID=containerID;copy.BranchName=branch;copy.PresetID=presetID;return &copy}
func truncate(v string,n int)string{if len(v)<=n{return v};return v[:n]+"...[truncated]"}
func finalizeGit(ctx context.Context,dir,base,taskID string)(string,[]string,[]string){
  warnings:=[]string{}
  run:=func(args ...string)(string,error){cmd:=exec.CommandContext(ctx,"git",append([]string{"-C",dir},args...)...);out,err:=cmd.CombinedOutput();return strings.TrimSpace(string(out)),err}
  status,err:=run("status","--porcelain");if err!=nil{warnings=append(warnings,"Could not inspect final Git status: "+err.Error());return "",warnings,nil}
  if status!=""{
    _,err=run("add","-A");if err==nil{_,err=run("commit","-m","agent: complete task "+taskID)}
    if err!=nil{warnings=append(warnings,"Automatic local commit failed: "+err.Error())}
  }
  head,headErr:=run("rev-parse","HEAD");if headErr!=nil{warnings=append(warnings,"Could not read resulting commit: "+headErr.Error());head=""}
  changed:=[]string{}
  if base!=""{out,e:=run("diff","--name-only",base+"...HEAD");if e==nil&&out!=""{changed=strings.Split(out,"\n")}}
  if len(changed)==0{out,e:=run("show","--pretty=format:","--name-only","HEAD");if e==nil&&out!=""{changed=strings.Split(out,"\n")}}
  return head,warnings,changed
}

// Tool policy APIs.
func(s *Server) handleListToolPolicies(w http.ResponseWriter,r *http.Request){if s.toolPolicies==nil{s.jsonError(w,"tool policy store unavailable",503);return};s.jsonOK(w,map[string]interface{}{"tools":s.toolPolicies.List()})}
func(s *Server) handleUpdateToolPolicy(w http.ResponseWriter,r *http.Request){
  if s.toolPolicies==nil{s.jsonError(w,"tool policy store unavailable",503);return}
  var body struct{Enabled *bool `json:"enabled"`;Policy tools.Policy `json:"policy"`}
  if err:=json.NewDecoder(r.Body).Decode(&body);err!=nil{s.jsonError(w,"invalid request",400);return}
  current,ok:=s.toolPolicies.Get(r.PathValue("id"));if !ok{s.jsonError(w,"tool not found",404);return}
  enabled:=current.Enabled;if body.Enabled!=nil{enabled=*body.Enabled}
  updated,err:=s.toolPolicies.SetPolicy(current.ID,enabled,body.Policy);if err!=nil{s.jsonError(w,err.Error(),400);return};s.jsonOK(w,map[string]interface{}{"tool":updated})
}
func(s *Server) handleListApprovals(w http.ResponseWriter,r *http.Request){
  if s.toolPolicies==nil{s.jsonError(w,"approval store unavailable",503);return}
  rows:=s.toolPolicies.ListApprovals(r.URL.Query().Get("status"))
  if getUserRole(r.Context())!="admin"{
    visible:=rows[:0];for _,a:=range rows{t,ok:=s.tasks.Get(a.TaskID);if ok&&t.OwnerID==getUserName(r.Context()){visible=append(visible,a)}};rows=visible
  }
  s.jsonOK(w,map[string]interface{}{"approvals":rows})
}
func(s *Server) handleRequestApproval(w http.ResponseWriter,r *http.Request){
  if s.toolPolicies==nil||s.tasks==nil{s.jsonError(w,"approval service unavailable",503);return}
  var req struct{TaskID string `json:"taskId"`;ToolID string `json:"toolId"`;ResourceType string `json:"resourceType"`;ResourceID string `json:"resourceId"`;ArgumentsDigest string `json:"argumentsDigest"`;Summary string `json:"summary"`;AgentSessionID string `json:"agentSessionId"`}
  if err:=json.NewDecoder(r.Body).Decode(&req);err!=nil{s.jsonError(w,"invalid request",400);return}
  t,ok:=s.tasks.Get(req.TaskID);if !ok||t.OwnerID!=getUserName(r.Context()){s.jsonError(w,"task not found",404);return}
  d,ok:=s.toolPolicies.Get(req.ToolID);if !ok||!d.Enabled||d.Policy!=tools.ApprovalRequired{s.jsonError(w,"tool does not require or allow approval",403);return}
  a:=tools.Approval{ID:task.NewID(),TaskID:req.TaskID,AgentSessionID:req.AgentSessionID,ToolID:req.ToolID,ResourceType:req.ResourceType,ResourceID:req.ResourceID,ArgumentsDigest:req.ArgumentsDigest,Summary:req.Summary,RequestedBy:getUserName(r.Context()),ExpiresAt:time.Now().UTC().Add(10*time.Minute)}
  if err:=s.toolPolicies.CreateApproval(a);err!=nil{s.jsonError(w,err.Error(),500);return};s.jsonOK(w,map[string]interface{}{"approval":a})
}
func(s *Server) reviewApproval(w http.ResponseWriter,r *http.Request,status string){
  if s.toolPolicies==nil{s.jsonError(w,"approval store unavailable",503);return}
  a,err:=s.toolPolicies.ReviewApproval(r.PathValue("id"),getUserName(r.Context()),status);if err!=nil{s.jsonError(w,err.Error(),409);return};s.jsonOK(w,map[string]interface{}{"approval":a})
}
func(s *Server) handleApprove(w http.ResponseWriter,r *http.Request){s.reviewApproval(w,r,"approved")}
func(s *Server) handleDeny(w http.ResponseWriter,r *http.Request){s.reviewApproval(w,r,"denied")}

// Agent session and context APIs.
func(s *Server) handleRegisterAgent(w http.ResponseWriter,r *http.Request){
  if s.coordination==nil{s.jsonError(w,"coordination store unavailable",503);return}
  var req struct{ProjectID string `json:"projectId"`;TaskID string `json:"taskId"`;WorkstreamID string `json:"workstreamId"`;AgentLabel string `json:"agentLabel"`;Branch string `json:"branch"`;Commit string `json:"commit"`}
  if err:=json.NewDecoder(r.Body).Decode(&req);err!=nil{s.jsonError(w,"invalid request",400);return}
  if req.TaskID!=""{t,ok:=s.tasks.Get(req.TaskID);if !ok||t.OwnerID!=getUserName(r.Context())||t.ProjectID!=req.ProjectID{s.jsonError(w,"task not found",404);return}}
  session,err:=s.coordination.RegisterOwned(getUserName(r.Context()),req.ProjectID,req.TaskID,req.WorkstreamID,req.AgentLabel,req.Branch,req.Commit);if err!=nil{s.jsonError(w,err.Error(),400);return}
  latest,messages,agents:=s.coordination.LatestContext(req.ProjectID,req.TaskID)
  s.jsonOK(w,map[string]interface{}{"session":session,"contextBundle":map[string]interface{}{"snapshot":latest,"recentMessages":messages,"agents":agents}})
}
func(s *Server) ownedSession(r *http.Request)(coordination.Session,bool){
  v,ok:=s.coordination.GetSession(r.PathValue("id"));if !ok||v.OwnerID!=getUserName(r.Context()){return coordination.Session{},false};return v,true
}
func(s *Server) handleAgentHeartbeat(w http.ResponseWriter,r *http.Request){
  if s.coordination==nil{s.jsonError(w,"coordination store unavailable",503);return};v,ok:=s.ownedSession(r);if !ok{s.jsonError(w,"agent session not found",404);return}
  var req struct{Branch string `json:"branch"`;Commit string `json:"commit"`};_=json.NewDecoder(r.Body).Decode(&req)
  updated,err:=s.coordination.Heartbeat(v.ID,req.Branch,req.Commit);if err!=nil{s.jsonError(w,err.Error(),400);return};s.jsonOK(w,map[string]interface{}{"session":updated})
}
func(s *Server) handleResumeAgent(w http.ResponseWriter,r *http.Request){
  if s.coordination==nil{s.jsonError(w,"coordination store unavailable",503);return};v,ok:=s.ownedSession(r);if !ok{s.jsonError(w,"agent session not found",404);return}
  latest,messages,agents:=s.coordination.LatestContext(v.ProjectID,v.TaskID)
  s.jsonOK(w,map[string]interface{}{"session":v,"contextBundle":map[string]interface{}{"snapshot":latest,"recentMessages":messages,"agents":agents,"bootstrap":"Read AGENTS.md, REWORK_PLAN.md, ARCHITECTURE.md, docs/agents/COORDINATION.md and docs/agents/WORKSTREAMS.md. Verify the current branch and commit before continuing."}})
}
func(s *Server) handleAgentContext(w http.ResponseWriter,r *http.Request){s.handleResumeAgent(w,r)}
func(s *Server) handleCreateHandoff(w http.ResponseWriter,r *http.Request){
  if s.coordination==nil{s.jsonError(w,"coordination store unavailable",503);return};session,ok:=s.ownedSession(r);if !ok{s.jsonError(w,"agent session not found",404);return}
  var snap coordination.Snapshot;if err:=json.NewDecoder(r.Body).Decode(&snap);err!=nil{s.jsonError(w,"invalid handoff body",400);return}
  snap.ProjectID=session.ProjectID;snap.TaskID=session.TaskID;snap.SessionID=session.ID;snap.WorkstreamID=session.WorkstreamID
  if snap.Branch==""{snap.Branch=session.Branch};if snap.Commit==""{snap.Commit=session.Commit}
  result,err:=s.coordination.CreateHandoff(snap);if err!=nil{s.jsonError(w,err.Error(),400);return};s.jsonOK(w,map[string]interface{}{"snapshot":result})
}
func(s *Server) visibleProject(projectID string,user string,admin bool)bool{
  if admin{return true}
  for _,t:=range s.tasks.List(user){if t.ProjectID==projectID{return true}}
  for _,a:=range s.coordination.ListSessions(projectID){if a.OwnerID==user{return true}}
  return false
}
func(s *Server) handleListProjectAgents(w http.ResponseWriter,r *http.Request){
  project:=r.PathValue("projectID");if s.coordination==nil||!s.visibleProject(project,getUserName(r.Context()),getUserRole(r.Context())=="admin"){s.jsonError(w,"project not found",404);return}
  out:=[]coordination.Session{};for _,v:=range s.coordination.ListSessions(project){if getUserRole(r.Context())=="admin"||v.OwnerID==getUserName(r.Context()){out=append(out,v)}}
  s.jsonOK(w,map[string]interface{}{"agents":out})
}
func(s *Server) handleProjectContext(w http.ResponseWriter,r *http.Request){
  project:=r.PathValue("projectID");if s.coordination==nil||!s.visibleProject(project,getUserName(r.Context()),getUserRole(r.Context())=="admin"){s.jsonError(w,"project not found",404);return}
  latest,messages,agents:=s.coordination.LatestContext(project,"")
  if getUserRole(r.Context())!="admin"{filtered:=messages[:0];for _,m:=range messages{sess,ok:=s.coordination.GetSession(m.SenderSessionID);if ok&&sess.OwnerID==getUserName(r.Context()){filtered=append(filtered,m)}};messages=filtered}
  s.jsonOK(w,map[string]interface{}{"snapshot":latest,"messages":messages,"agents":agents})
}
func(s *Server) handleReadAgentMessages(w http.ResponseWriter,r *http.Request){
  project:=r.PathValue("projectID");if s.coordination==nil||!s.visibleProject(project,getUserName(r.Context()),getUserRole(r.Context())=="admin"){s.jsonError(w,"project not found",404);return}
  after,_:=strconv.ParseInt(r.URL.Query().Get("after"),10,64);taskID:=r.URL.Query().Get("taskId")
  out:=s.coordination.Messages(project,taskID,after,200)
  if getUserRole(r.Context())!="admin"{filtered:=out[:0];for _,m:=range out{sess,ok:=s.coordination.GetSession(m.SenderSessionID);if ok&&sess.OwnerID==getUserName(r.Context()){filtered=append(filtered,m)}};out=filtered}
  s.jsonOK(w,map[string]interface{}{"messages":out})
}
func(s *Server) handleSendAgentMessage(w http.ResponseWriter,r *http.Request){
  if s.coordination==nil{s.jsonError(w,"coordination store unavailable",503);return}
  var req struct{TaskID string `json:"taskId"`;Channel string `json:"channel"`;SenderSessionID string `json:"senderSessionId"`;RecipientSessionID string `json:"recipientSessionId"`;Type string `json:"type"`;Body string `json:"body"`;CorrelationID string `json:"correlationId"`}
  if err:=json.NewDecoder(r.Body).Decode(&req);err!=nil{s.jsonError(w,"invalid message",400);return}
  session,ok:=s.coordination.GetSession(req.SenderSessionID);if !ok||session.OwnerID!=getUserName(r.Context())||session.ProjectID!=r.PathValue("projectID"){s.jsonError(w,"sender session not found",404);return}
  if req.RecipientSessionID!=""{recipient,ok:=s.coordination.GetSession(req.RecipientSessionID);if !ok||recipient.ProjectID!=session.ProjectID{s.jsonError(w,"recipient session not found",404);return}}
  msg,err:=s.coordination.SendMessage(coordination.Message{ProjectID:session.ProjectID,TaskID:req.TaskID,Channel:req.Channel,SenderSessionID:req.SenderSessionID,RecipientSessionID:req.RecipientSessionID,Type:req.Type,Body:req.Body,CorrelationID:req.CorrelationID})
  if err!=nil{s.jsonError(w,err.Error(),400);return};s.jsonOK(w,map[string]interface{}{"message":msg})
}
func(s *Server) handleClaimWorkItem(w http.ResponseWriter,r *http.Request){
  if s.coordination==nil{s.jsonError(w,"coordination store unavailable",503);return}
  var req struct{ProjectID string `json:"projectId"`;SessionID string `json:"sessionId"`;Resource string `json:"resource"`;Kind string `json:"kind"`;TTLSeconds int `json:"ttlSeconds"`}
  if err:=json.NewDecoder(r.Body).Decode(&req);err!=nil{s.jsonError(w,"invalid claim",400);return}
  session,ok:=s.coordination.GetSession(req.SessionID);if !ok||session.OwnerID!=getUserName(r.Context())||session.ProjectID!=req.ProjectID{s.jsonError(w,"session not found",404);return}
  ttl:=time.Duration(req.TTLSeconds)*time.Second;if ttl==0{ttl=15*time.Minute}
  lease,err:=s.coordination.AcquireLease(req.ProjectID,req.SessionID,req.Resource,req.Kind,ttl);if err!=nil{s.jsonError(w,err.Error(),http.StatusConflict);return};s.jsonOK(w,map[string]interface{}{"lease":lease})
}
func(s *Server) handleReleaseWorkItem(w http.ResponseWriter,r *http.Request){
  if s.coordination==nil{s.jsonError(w,"coordination store unavailable",503);return}
  var req struct{SessionID string `json:"sessionId"`};if err:=json.NewDecoder(r.Body).Decode(&req);err!=nil{s.jsonError(w,"sessionId required",400);return}
  session,ok:=s.coordination.GetSession(req.SessionID);if !ok||session.OwnerID!=getUserName(r.Context()){s.jsonError(w,"session not found",404);return}
  if err:=s.coordination.ReleaseLease(r.PathValue("id"),req.SessionID);err!=nil{s.jsonError(w,err.Error(),403);return};s.jsonOK(w,map[string]string{"status":"released"})
}
