package coordination

import (
  "crypto/rand"
  "encoding/hex"
  "encoding/json"
  "errors"
  "os"
  "path/filepath"
  "sort"
  "sync"
  "time"
)

type Session struct {
  ID string `json:"id"`
  ProjectID string `json:"projectId"`
  TaskID string `json:"taskId,omitempty"`
  WorkstreamID string `json:"workstreamId,omitempty"`
  AgentLabel string `json:"agentLabel"`
  Branch string `json:"branch,omitempty"`
  Commit string `json:"commit,omitempty"`
  Status string `json:"status"`
  LastHeartbeatAt time.Time `json:"lastHeartbeatAt"`
  CreatedAt time.Time `json:"createdAt"`
}

type Message struct {
  ID string `json:"id"`
  Cursor int64 `json:"cursor"`
  ProjectID string `json:"projectId"`
  TaskID string `json:"taskId,omitempty"`
  Channel string `json:"channel"`
  SenderSessionID string `json:"senderSessionId"`
  RecipientSessionID string `json:"recipientSessionId,omitempty"`
  Type string `json:"type"`
  Body string `json:"body"`
  CorrelationID string `json:"correlationId,omitempty"`
  CreatedAt time.Time `json:"createdAt"`
}

type Snapshot struct {
  ID string `json:"id"`
  ProjectID string `json:"projectId"`
  TaskID string `json:"taskId,omitempty"`
  SessionID string `json:"sessionId"`
  WorkstreamID string `json:"workstreamId,omitempty"`
  RepositoryURL string `json:"repositoryUrl,omitempty"`
  Branch string `json:"branch,omitempty"`
  Commit string `json:"commit,omitempty"`
  Goal string `json:"goal"`
  Summary string `json:"summary"`
  Constraints []string `json:"constraints,omitempty"`
  Decisions []string `json:"decisions,omitempty"`
  Completed []string `json:"completed,omitempty"`
  InProgress []string `json:"inProgress,omitempty"`
  ChangedPaths []string `json:"changedPaths,omitempty"`
  Checks []string `json:"checks,omitempty"`
  Blockers []string `json:"blockers,omitempty"`
  NextActions []string `json:"nextActions,omitempty"`
  PreviousSnapshotID string `json:"previousSnapshotId,omitempty"`
  MessageCursor int64 `json:"messageCursor"`
  CreatedAt time.Time `json:"createdAt"`
}

type Lease struct {
  ID string `json:"id"`
  ProjectID string `json:"projectId"`
  SessionID string `json:"sessionId"`
  Resource string `json:"resource"`
  Kind string `json:"kind"`
  ExpiresAt time.Time `json:"expiresAt"`
  CreatedAt time.Time `json:"createdAt"`
}

type diskState struct {
  Sessions map[string]Session `json:"sessions"`
  Messages []Message `json:"messages"`
  Snapshots []Snapshot `json:"snapshots"`
  Leases map[string]Lease `json:"leases"`
  Cursor int64 `json:"cursor"`
}
type Store struct { mu sync.RWMutex; path string; data diskState }

func id()string{b:=make([]byte,16);if _,e:=rand.Read(b);e!=nil{return time.Now().UTC().Format("20060102150405.000000000")};return hex.EncodeToString(b)}
func OpenStore(path string)(*Store,error){
  if path==""{return nil,errors.New("coordination store path required")}
  s:=&Store{path:path,data:diskState{Sessions:map[string]Session{},Leases:map[string]Lease{}}}
  b,e:=os.ReadFile(path);if e==nil{if e=json.Unmarshal(b,&s.data);e!=nil{return nil,e}}else if !os.IsNotExist(e){return nil,e}
  if s.data.Sessions==nil{s.data.Sessions=map[string]Session{}};if s.data.Leases==nil{s.data.Leases=map[string]Lease{}}
  return s,nil
}
func(s *Store)saveLocked()error{if e:=os.MkdirAll(filepath.Dir(s.path),0700);e!=nil{return e};b,e:=json.MarshalIndent(s.data,"","  ");if e!=nil{return e};tmp:=s.path+".tmp";if e=os.WriteFile(tmp,b,0600);e!=nil{return e};return os.Rename(tmp,s.path)}
func(s *Store)Register(project,task,workstream,label,branch,commit string)(Session,error){
  if project==""||label==""{return Session{},errors.New("projectId and agentLabel are required")}
  s.mu.Lock();defer s.mu.Unlock();now:=time.Now().UTC();v:=Session{ID:id(),ProjectID:project,TaskID:task,WorkstreamID:workstream,AgentLabel:label,Branch:branch,Commit:commit,Status:"active",CreatedAt:now,LastHeartbeatAt:now};s.data.Sessions[v.ID]=v;if e:=s.saveLocked();e!=nil{return Session{},e};return v,nil
}
func(s *Store)GetSession(sessionID string)(Session,bool){s.mu.RLock();defer s.mu.RUnlock();v,ok:=s.data.Sessions[sessionID];return v,ok}
func(s *Store)Heartbeat(sessionID,branch,commit string)(Session,error){s.mu.Lock();defer s.mu.Unlock();v,ok:=s.data.Sessions[sessionID];if !ok{return Session{},os.ErrNotExist};v.LastHeartbeatAt=time.Now().UTC();v.Status="active";if branch!=""{v.Branch=branch};if commit!=""{v.Commit=commit};s.data.Sessions[sessionID]=v;if e:=s.saveLocked();e!=nil{return Session{},e};return v,nil}
func(s *Store)ListSessions(project string)[]Session{s.mu.RLock();defer s.mu.RUnlock();out:=[]Session{};for _,v:=range s.data.Sessions{if project==""||v.ProjectID==project{out=append(out,v)}};sort.Slice(out,func(i,j int)bool{return out[i].LastHeartbeatAt.After(out[j].LastHeartbeatAt)});return out}
func(s *Store)SendMessage(v Message)(Message,error){
  if v.ProjectID==""||v.Channel==""||v.SenderSessionID==""||v.Type==""{return Message{},errors.New("project, channel, sender, and type required")}
  s.mu.Lock();defer s.mu.Unlock();if _,ok:=s.data.Sessions[v.SenderSessionID];!ok{return Message{},errors.New("sender session not found")}
  s.data.Cursor++;v.ID=id();v.Cursor=s.data.Cursor;v.CreatedAt=time.Now().UTC();s.data.Messages=append(s.data.Messages,v);if e:=s.saveLocked();e!=nil{return Message{},e};return v,nil
}
func(s *Store)Messages(project,task string,after int64,limit int)[]Message{s.mu.RLock();defer s.mu.RUnlock();if limit<=0||limit>500{limit=100};out:=[]Message{};for _,m:=range s.data.Messages{if m.Cursor>after&&m.ProjectID==project&&(task==""||m.TaskID==task){out=append(out,m);if len(out)>=limit{break}}};return out}
func(s *Store)CreateHandoff(v Snapshot)(Snapshot,error){
  if v.ProjectID==""||v.SessionID==""||v.Goal==""{return Snapshot{},errors.New("projectId, sessionId, and goal are required")}
  s.mu.Lock();defer s.mu.Unlock();sess,ok:=s.data.Sessions[v.SessionID];if !ok||sess.ProjectID!=v.ProjectID{return Snapshot{},errors.New("session does not belong to project")}
  v.ID=id();v.CreatedAt=time.Now().UTC();v.MessageCursor=s.data.Cursor
  for i:=len(s.data.Snapshots)-1;i>=0;i--{p:=s.data.Snapshots[i];if p.ProjectID==v.ProjectID&&(v.TaskID==""||p.TaskID==v.TaskID){v.PreviousSnapshotID=p.ID;break}}
  s.data.Snapshots=append(s.data.Snapshots,v);if e:=s.saveLocked();e!=nil{return Snapshot{},e};return v,nil
}
func(s *Store)LatestContext(project,task string)(*Snapshot,[]Message,[]Session){
  s.mu.RLock();defer s.mu.RUnlock();var latest *Snapshot
  for i:=len(s.data.Snapshots)-1;i>=0;i--{v:=s.data.Snapshots[i];if v.ProjectID==project&&(task==""||v.TaskID==task){c:=v;latest=&c;break}}
  cursor:=int64(0);if latest!=nil{cursor=latest.MessageCursor}
  msgs:=[]Message{};for _,m:=range s.data.Messages{if m.ProjectID==project&&(task==""||m.TaskID==task)&&m.Cursor>cursor{msgs=append(msgs,m)}}
  agents:=[]Session{};for _,a:=range s.data.Sessions{if a.ProjectID==project&&(task==""||a.TaskID==task){agents=append(agents,a)}}
  sort.Slice(agents,func(i,j int)bool{return agents[i].LastHeartbeatAt.After(agents[j].LastHeartbeatAt)})
  return latest,msgs,agents
}
func(s *Store)AcquireLease(project,session,resource,kind string,ttl time.Duration)(Lease,error){
  if project==""||session==""||resource==""{return Lease{},errors.New("project, session, and resource required")}
  if ttl<=0||ttl>24*time.Hour{ttl=15*time.Minute}
  s.mu.Lock();defer s.mu.Unlock();if a,ok:=s.data.Sessions[session];!ok||a.ProjectID!=project{return Lease{},errors.New("session does not belong to project")}
  now:=time.Now().UTC();for k,v:=range s.data.Leases{if !v.ExpiresAt.After(now){delete(s.data.Leases,k);continue};if v.ProjectID==project&&v.Resource==resource&&v.Kind==kind&&v.SessionID!=session{return Lease{},errors.New("resource is already leased")}}
  v:=Lease{ID:id(),ProjectID:project,SessionID:session,Resource:resource,Kind:kind,CreatedAt:now,ExpiresAt:now.Add(ttl)};s.data.Leases[v.ID]=v;if e:=s.saveLocked();e!=nil{return Lease{},e};return v,nil
}
func(s *Store)ReleaseLease(id,session string)error{s.mu.Lock();defer s.mu.Unlock();v,ok:=s.data.Leases[id];if !ok{return nil};if v.SessionID!=session{return errors.New("lease owner mismatch")};delete(s.data.Leases,id);return s.saveLocked()}
