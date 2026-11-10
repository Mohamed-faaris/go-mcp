package task

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

type diskState struct {
  Tasks map[string]*Task `json:"tasks"`
  Reports map[string]Report `json:"reports"`
}

type Store struct {
  mu sync.RWMutex
  path string
  state diskState
}

func OpenStore(path string) (*Store, error) {
  if path == "" { return nil, errors.New("task store path is required") }
  s := &Store{path:path, state:diskState{Tasks:map[string]*Task{}, Reports:map[string]Report{}}}
  data, err := os.ReadFile(path)
  if err == nil {
    if err := json.Unmarshal(data, &s.state); err != nil { return nil, err }
  } else if !os.IsNotExist(err) { return nil, err }
  if s.state.Tasks == nil { s.state.Tasks = map[string]*Task{} }
  if s.state.Reports == nil { s.state.Reports = map[string]Report{} }
  return s, nil
}

func NewID() string {
  b:=make([]byte,16)
  if _,err:=rand.Read(b); err!=nil { return time.Now().UTC().Format("20060102150405.000000000") }
  return hex.EncodeToString(b)
}

func cloneTask(t *Task) *Task { if t==nil{return nil}; b,_:=json.Marshal(t); var c Task; _=json.Unmarshal(b,&c); return &c }

func (s *Store) saveLocked() error {
  if err:=os.MkdirAll(filepath.Dir(s.path),0700); err!=nil{return err}
  b,err:=json.MarshalIndent(s.state,"","  "); if err!=nil{return err}
  tmp:=s.path+".tmp"
  if err=os.WriteFile(tmp,b,0600); err!=nil{return err}
  return os.Rename(tmp,s.path)
}

func (s *Store) Create(t *Task) error {
  if t==nil || t.ID=="" { return errors.New("invalid task") }
  s.mu.Lock(); defer s.mu.Unlock()
  if _,ok:=s.state.Tasks[t.ID]; ok { return errors.New("task already exists") }
  s.state.Tasks[t.ID]=cloneTask(t)
  if err:=s.saveLocked(); err!=nil { delete(s.state.Tasks,t.ID); return err }
  return nil
}

func (s *Store) Get(id string) (*Task,bool) {
  s.mu.RLock(); defer s.mu.RUnlock()
  t,ok:=s.state.Tasks[id]; return cloneTask(t),ok
}

func (s *Store) Update(id string, mutate func(*Task) error) (*Task,error) {
  s.mu.Lock(); defer s.mu.Unlock()
  current,ok:=s.state.Tasks[id]; if !ok{return nil,os.ErrNotExist}
  next:=cloneTask(current)
  if err:=mutate(next); err!=nil{return nil,err}
  s.state.Tasks[id]=next
  if err:=s.saveLocked(); err!=nil{s.state.Tasks[id]=current;return nil,err}
  return cloneTask(next),nil
}

func (s *Store) List(owner string) []*Task {
  s.mu.RLock(); defer s.mu.RUnlock()
  out:=make([]*Task,0,len(s.state.Tasks))
  for _,t:=range s.state.Tasks { if owner=="" || t.OwnerID==owner { out=append(out,cloneTask(t)) } }
  sort.Slice(out,func(i,j int)bool{return out[i].CreatedAt.After(out[j].CreatedAt)})
  return out
}

func (s *Store) SaveReport(report Report) error {
  s.mu.Lock(); defer s.mu.Unlock()
  s.state.Reports[report.TaskID]=report
  return s.saveLocked()
}

func (s *Store) GetReport(taskID string) (Report,bool) {
  s.mu.RLock(); defer s.mu.RUnlock()
  r,ok:=s.state.Reports[taskID]; return r,ok
}
