package tools

import (
  "encoding/json"
  "errors"
  "os"
  "path/filepath"
  "sort"
  "sync"
  "time"
)

type Policy string
const (
  Automatic Policy = "automatic"
  ApprovalRequired Policy = "approval_required"
  Disabled Policy = "disabled"
  HardDenied Policy = "hard_denied"
)

type Definition struct {
  ID string `json:"id"`
  Name string `json:"name"`
  Description string `json:"description"`
  Category string `json:"category"`
  Scope string `json:"scope"`
  Risk string `json:"risk"`
  Source string `json:"source"`
  Enabled bool `json:"enabled"`
  Policy Policy `json:"policy"`
  HardDenied bool `json:"hardDenied"`
  UpdatedAt time.Time `json:"updatedAt"`
}

type Approval struct {
  ID string `json:"id"`
  TaskID string `json:"taskId"`
  AgentSessionID string `json:"agentSessionId,omitempty"`
  ToolID string `json:"toolId"`
  ResourceType string `json:"resourceType"`
  ResourceID string `json:"resourceId"`
  ArgumentsDigest string `json:"argumentsDigest"`
  Summary string `json:"summary"`
  Status string `json:"status"`
  RequestedBy string `json:"requestedBy"`
  ReviewedBy string `json:"reviewedBy,omitempty"`
  CreatedAt time.Time `json:"createdAt"`
  ExpiresAt time.Time `json:"expiresAt"`
  ReviewedAt *time.Time `json:"reviewedAt,omitempty"`
}

type state struct {
  Tools map[string]Definition `json:"tools"`
  Approvals map[string]Approval `json:"approvals"`
}

type Store struct { mu sync.RWMutex; path string; data state }

func defaults() map[string]Definition {
  now:=time.Now().UTC()
  rows:=[]Definition{
    {ID:"fs.read",Name:"read_file",Description:"Read files inside the assigned project workspace.",Category:"filesystem",Scope:"task-workspace",Risk:"low",Source:"native",Enabled:true,Policy:Automatic,UpdatedAt:now},
    {ID:"fs.write",Name:"write_file",Description:"Write files inside the assigned project workspace.",Category:"filesystem",Scope:"task-workspace",Risk:"medium",Source:"native",Enabled:true,Policy:Automatic,UpdatedAt:now},
    {ID:"fs.search",Name:"search_files",Description:"Search filenames and file contents inside the workspace.",Category:"filesystem",Scope:"task-workspace",Risk:"low",Source:"native",Enabled:true,Policy:Automatic,UpdatedAt:now},
    {ID:"process.run",Name:"run_command",Description:"Run bounded commands inside the assigned task container.",Category:"terminal",Scope:"task-container",Risk:"medium",Source:"native",Enabled:true,Policy:Automatic,UpdatedAt:now},
    {ID:"process.manage",Name:"manage_process",Description:"Inspect and terminate processes inside the assigned task container.",Category:"process",Scope:"task-container",Risk:"medium",Source:"native",Enabled:true,Policy:Automatic,UpdatedAt:now},
    {ID:"git.local",Name:"git_local",Description:"Inspect diffs and create local branches or commits.",Category:"git",Scope:"task-repository",Risk:"low",Source:"native",Enabled:true,Policy:Automatic,UpdatedAt:now},
    {ID:"git.push",Name:"git_push",Description:"Push the task branch to its connected repository through the host credential broker.",Category:"git",Scope:"assigned-repository",Risk:"high",Source:"native",Enabled:true,Policy:ApprovalRequired,UpdatedAt:now},
    {ID:"container.create",Name:"create_task_container",Description:"Request another managed task container.",Category:"container",Scope:"manager-mediated",Risk:"high",Source:"native",Enabled:true,Policy:ApprovalRequired,UpdatedAt:now},
    {ID:"container.delete.other",Name:"delete_other_container",Description:"Request deletion of another managed task container.",Category:"container",Scope:"manager-mediated",Risk:"critical",Source:"native",Enabled:true,Policy:ApprovalRequired,UpdatedAt:now},
    {ID:"host.files",Name:"host_filesystem",Description:"Direct access to the host filesystem.",Category:"host",Scope:"host",Risk:"critical",Source:"native",Enabled:false,Policy:HardDenied,HardDenied:true,UpdatedAt:now},
    {ID:"host.podman",Name:"podman_socket",Description:"Direct access to the host Podman socket.",Category:"host",Scope:"host",Risk:"critical",Source:"native",Enabled:false,Policy:HardDenied,HardDenied:true,UpdatedAt:now},
  }
  out:=map[string]Definition{}; for _,r:=range rows{out[r.ID]=r}; return out
}

func OpenStore(path string)(*Store,error){
  s:=&Store{path:path,data:state{Tools:defaults(),Approvals:map[string]Approval{}}}
  if path=="" {return nil,errors.New("tool store path required")}
  b,err:=os.ReadFile(path); if err==nil {if err=json.Unmarshal(b,&s.data);err!=nil{return nil,err}; if s.data.Tools==nil{s.data.Tools=defaults()};if s.data.Approvals==nil{s.data.Approvals=map[string]Approval{}}} else if !os.IsNotExist(err){return nil,err} else if e:=s.save();e!=nil{return nil,e}
  for id,d:=range defaults(){if _,ok:=s.data.Tools[id];!ok{s.data.Tools[id]=d}}
  return s,nil
}
func(s *Store) saveLocked()error{if e:=os.MkdirAll(filepath.Dir(s.path),0700);e!=nil{return e};b,e:=json.MarshalIndent(s.data,"","  ");if e!=nil{return e};tmp:=s.path+".tmp";if e=os.WriteFile(tmp,b,0600);e!=nil{return e};return os.Rename(tmp,s.path)}
func(s *Store) save()error{s.mu.Lock();defer s.mu.Unlock();return s.saveLocked()}
func(s *Store) List()[]Definition{s.mu.RLock();defer s.mu.RUnlock();out:=make([]Definition,0,len(s.data.Tools));for _,v:=range s.data.Tools{out=append(out,v)};sort.Slice(out,func(i,j int)bool{return out[i].Category==out[j].Category&&out[i].Name<out[j].Name || out[i].Category<out[j].Category});return out}
func(s *Store) Get(id string)(Definition,bool){s.mu.RLock();defer s.mu.RUnlock();v,ok:=s.data.Tools[id];return v,ok}
func(s *Store) SetPolicy(id string,enabled bool,policy Policy)(Definition,error){
  if policy!=Automatic&&policy!=ApprovalRequired&&policy!=Disabled{return Definition{},errors.New("invalid tool policy")}
  s.mu.Lock();defer s.mu.Unlock();v,ok:=s.data.Tools[id];if !ok{return Definition{},os.ErrNotExist};if v.HardDenied{return Definition{},errors.New("hard-denied tools cannot be changed")}
  v.Enabled=enabled;v.Policy=policy;v.UpdatedAt=time.Now().UTC();s.data.Tools[id]=v;if e:=s.saveLocked();e!=nil{return Definition{},e};return v,nil
}
func(s *Store) CreateApproval(a Approval)error{
  s.mu.Lock();defer s.mu.Unlock();if a.ID==""{return errors.New("approval id required")};if a.CreatedAt.IsZero(){a.CreatedAt=time.Now().UTC()};if a.ExpiresAt.IsZero(){a.ExpiresAt=time.Now().UTC().Add(10*time.Minute)};a.Status="pending";s.data.Approvals[a.ID]=a;return s.saveLocked()
}
func(s *Store) ListApprovals(status string)[]Approval{s.mu.RLock();defer s.mu.RUnlock();out:=[]Approval{};for _,a:=range s.data.Approvals{if status==""||a.Status==status{out=append(out,a)}};sort.Slice(out,func(i,j int)bool{return out[i].CreatedAt.After(out[j].CreatedAt)});return out}
func(s *Store) ReviewApproval(id,reviewer,status string)(Approval,error){
  if status!="approved"&&status!="denied"{return Approval{},errors.New("invalid approval decision")}
  s.mu.Lock();defer s.mu.Unlock();a,ok:=s.data.Approvals[id];if !ok{return Approval{},os.ErrNotExist};if a.Status!="pending"{return Approval{},errors.New("approval is no longer pending")};now:=time.Now().UTC();if now.After(a.ExpiresAt){a.Status="expired";s.data.Approvals[id]=a;_=s.saveLocked();return a,errors.New("approval expired")};a.Status=status;a.ReviewedBy=reviewer;a.ReviewedAt=&now;s.data.Approvals[id]=a;if e:=s.saveLocked();e!=nil{return Approval{},e};return a,nil
}
