package container

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeRuntime struct {
	items map[string]Info
	lastSpec Spec
	deleted []string
	stopped []string
	started []string
}
func newFakeRuntime() *fakeRuntime { return &fakeRuntime{items:map[string]Info{}} }
func (f *fakeRuntime) Create(_ context.Context,s Spec)(Info,error){
	f.lastSpec=s
	id:="runtime-1"
	labels:=map[string]string{"gomcp.managed":"true","gomcp.owner_id":s.OwnerID,"gomcp.task_id":s.TaskID,"gomcp.project_id":s.ProjectID,"gomcp.preset_id":s.PresetID}
	info:=Info{ID:id,Name:s.Name,Image:s.Image,State:"created",OwnerID:s.OwnerID,TaskID:s.TaskID,ProjectID:s.ProjectID,PresetID:s.PresetID,CreatedAt:time.Now(),Labels:labels}
	f.items[id]=info
	return info,nil
}
func(f *fakeRuntime)Start(_ context.Context,id string)error{f.started=append(f.started,id);i:=f.items[id];i.State="running";f.items[id]=i;return nil}
func(f *fakeRuntime)Stop(_ context.Context,id string,_ time.Duration)error{f.stopped=append(f.stopped,id);i:=f.items[id];i.State="stopped";f.items[id]=i;return nil}
func(f *fakeRuntime)Delete(_ context.Context,id string)error{f.deleted=append(f.deleted,id);delete(f.items,id);return nil}
func(f *fakeRuntime)Inspect(_ context.Context,id string)(Info,error){i,ok:=f.items[id];if !ok{return Info{},os.ErrNotExist};return i,nil}
func(f *fakeRuntime)ListManaged(context.Context)([]Info,error){out:=[]Info{};for _,i:=range f.items{out=append(out,i)};return out,nil}

func testManager(t *testing.T)(*Manager,*fakeRuntime,string){
	t.Helper()
	root:=t.TempDir()
	ws:=filepath.Join(root,"task-1")
	if err:=os.Mkdir(ws,0o700);err!=nil{t.Fatal(err)}
	rt:=newFakeRuntime()
	reg,err:=NewFileRegistry(filepath.Join(root,"state","registry.json"));if err!=nil{t.Fatal(err)}
	m,err:=NewManager(rt,reg,ManagerOptions{WorkspaceRoot:root,AllowedImages:[]string{"node:22"},Network:"agent-egress",DefaultLimits:ResourceLimits{MemoryBytes:2<<30,CPUCores:2,PidsLimit:512,Timeout:time.Hour}})
	if err!=nil{t.Fatal(err)}
	return m,rt,ws
}
func testRequest(ws string)CreateRequest{return CreateRequest{OwnerID:"user-1",TaskID:"task-1",ProjectID:"project-1",PresetID:"node-pnpm",Image:"node:22",Workspace:ws}}

func TestCreateAddsManagerOwnedSecurityIdentity(t *testing.T){
	m,rt,ws:=testManager(t);info,err:=m.CreateTaskContainer(context.Background(),testRequest(ws));if err!=nil{t.Fatal(err)}
	if info.OwnerID!="user-1"||info.TaskID!="task-1"{t.Fatalf("missing identity: %#v",info)}
	if !rt.lastSpec.ReadOnlyRoot||!rt.lastSpec.NoNewPrivileges||len(rt.lastSpec.DropCapabilities)!=1||rt.lastSpec.DropCapabilities[0]!="ALL"{t.Fatalf("unsafe runtime flags: %#v",rt.lastSpec)}
	if rt.lastSpec.Workspace!=ws{t.Fatalf("workspace mismatch: %s",rt.lastSpec.Workspace)}
}
func TestCreateRejectsUnknownImage(t *testing.T){m,_,ws:=testManager(t);req:=testRequest(ws);req.Image="evil/image:latest";if _,err:=m.CreateTaskContainer(context.Background(),req);!errors.Is(err,ErrImageNotAllowed){t.Fatalf("want ErrImageNotAllowed, got %v",err)}}
func TestCreateRejectsWorkspaceOutsideRoot(t *testing.T){m,_,_:=testManager(t);req:=testRequest(t.TempDir());if _,err:=m.CreateTaskContainer(context.Background(),req);err==nil{t.Fatal("expected outside workspace rejection")}}
func TestDeleteRequiresExactOwnerAndTask(t *testing.T){m,rt,ws:=testManager(t);info,err:=m.CreateTaskContainer(context.Background(),testRequest(ws));if err!=nil{t.Fatal(err)};if err:=m.DeleteOwned(context.Background(),"other-user","task-1",info.ID);!errors.Is(err,ErrNotOwned){t.Fatalf("want ErrNotOwned, got %v",err)};if len(rt.deleted)!=0{t.Fatal("unauthorized delete reached runtime")}}
func TestDeleteVerifiesRuntimeLabelsBeforeDelete(t *testing.T){m,rt,ws:=testManager(t);info,err:=m.CreateTaskContainer(context.Background(),testRequest(ws));if err!=nil{t.Fatal(err)};observed:=rt.items[info.ID];observed.Labels["gomcp.owner_id"]="attacker";rt.items[info.ID]=observed;if err:=m.DeleteOwned(context.Background(),"user-1","task-1",info.ID);err==nil{t.Fatal("expected identity mismatch")};if len(rt.deleted)!=0{t.Fatal("mismatched container was deleted")}}
func TestDeleteStopsRunningOwnedContainer(t *testing.T){m,rt,ws:=testManager(t);info,err:=m.CreateTaskContainer(context.Background(),testRequest(ws));if err!=nil{t.Fatal(err)};observed:=rt.items[info.ID];observed.State="running";rt.items[info.ID]=observed;if err:=m.DeleteOwned(context.Background(),"user-1","task-1",info.ID);err!=nil{t.Fatal(err)};if len(rt.stopped)!=1||len(rt.deleted)!=1{t.Fatalf("expected stop and delete, stopped=%v deleted=%v",rt.stopped,rt.deleted)}}
func TestStartStopAreOwnershipChecked(t *testing.T){m,rt,ws:=testManager(t);info,err:=m.CreateTaskContainer(context.Background(),testRequest(ws));if err!=nil{t.Fatal(err)};if err:=m.StartOwned(context.Background(),"user-1","task-1",info.ID);err!=nil{t.Fatal(err)};if err:=m.StopOwned(context.Background(),"user-1","task-1",info.ID,time.Second);err!=nil{t.Fatal(err)};if len(rt.started)!=1||len(rt.stopped)!=1{t.Fatalf("expected start+stop: %#v",rt)}}
func TestFileRegistryPersistsOwnershipWithPrivatePermissions(t *testing.T){path:=filepath.Join(t.TempDir(),"nested","registry.json");r,err:=NewFileRegistry(path);if err!=nil{t.Fatal(err)};info:=Info{ID:"runtime-1",OwnerID:"user-1",TaskID:"task-1"};if err:=r.Put(context.Background(),info);err!=nil{t.Fatal(err)};loaded,err:=NewFileRegistry(path);if err!=nil{t.Fatal(err)};got,err:=loaded.Get(context.Background(),"runtime-1");if err!=nil{t.Fatal(err)};if got.OwnerID!="user-1"{t.Fatalf("owner not persisted: %#v",got)};st,err:=os.Stat(path);if err!=nil{t.Fatal(err)};if st.Mode().Perm()!=0o600{t.Fatalf("expected 0600 registry, got %o",st.Mode().Perm())}}
