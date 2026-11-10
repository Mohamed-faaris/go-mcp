package podman

import (
	"context"
	"fmt"
	"strconv"
	"time"

	managed "github.com/Mohamed-faaris/go-mcp/internal/container"
)

// Runtime adapts the low-level Podman HTTP client to the restricted workload
// runtime interface used by the control-plane container manager.
type Runtime struct { client *Client }

func NewRuntime(client *Client) (*Runtime, error) {
	if client == nil { return nil, fmt.Errorf("podman runtime: client is required") }
	return &Runtime{client:client},nil
}

func (r *Runtime) Create(ctx context.Context, spec managed.Spec) (managed.Info,error) {
	env := make([]string,0,len(spec.Environment))
	for key,value := range spec.Environment { env=append(env,key+"="+value) }
	opts := CreateContainerOpts{
		Name:spec.Name, Image:spec.Image, ProjectDir:spec.Workspace,
		MemoryLimit:strconv.FormatInt(spec.Limits.MemoryBytes,10),
		CPULimit:strconv.FormatFloat(spec.Limits.CPUCores,'f',-1,64),
		PidsLimit:spec.Limits.PidsLimit, Network:spec.Network,
		OwnerID:spec.OwnerID, TaskID:spec.TaskID, ProjectID:spec.ProjectID,
		PresetID:spec.PresetID, ReadOnlyRootfs:spec.ReadOnlyRoot,
		NoNewPrivileges:spec.NoNewPrivileges, DropCapabilities:append([]string(nil),spec.DropCapabilities...),
		Env:env,
		// Devices and arbitrary MCP host mounts are deliberately not propagated.
	}
	id,err:=r.client.CreateContainer(ctx,opts)
	if err!=nil{return managed.Info{},err}
	info,err:=r.Inspect(ctx,id)
	if err!=nil {
		_ = r.client.RemoveContainer(context.Background(),id)
		return managed.Info{},fmt.Errorf("inspect newly created workload: %w",err)
	}
	return info,nil
}

func (r *Runtime) Start(ctx context.Context,id string) error { return r.client.StartContainer(ctx,id) }
func (r *Runtime) Stop(ctx context.Context,id string,timeout time.Duration) error {
	if timeout>0 {
		var cancel context.CancelFunc
		ctx,cancel=context.WithTimeout(ctx,timeout+5*time.Second)
		defer cancel()
	}
	return r.client.StopContainer(ctx,id)
}
func (r *Runtime) Delete(ctx context.Context,id string) error { return r.client.RemoveContainer(ctx,id) }

func (r *Runtime) Inspect(ctx context.Context,id string)(managed.Info,error) {
	item,err:=r.client.InspectContainer(ctx,id)
	if err!=nil{return managed.Info{},err}
	return infoFromInspect(item),nil
}

func (r *Runtime) ListManaged(ctx context.Context)([]managed.Info,error) {
	items,err:=r.client.ListContainers(ctx)
	if err!=nil{return nil,err}
	out:=make([]managed.Info,0,len(items))
	for _,item:=range items {
		if item.Labels==nil || item.Labels["gomcp.managed"]!="true" { continue }
		info:=managed.Info{
			ID:item.ID, Image:item.Image, State:item.State, Labels:cloneStringMap(item.Labels),
			OwnerID:item.Labels["gomcp.owner_id"], TaskID:item.Labels["gomcp.task_id"],
			ProjectID:item.Labels["gomcp.project_id"], PresetID:item.Labels["gomcp.preset_id"],
		}
		if len(item.Names)>0 { info.Name=item.Names[0] }
		if item.Created!=0 { info.CreatedAt=time.Unix(int64(item.Created),0).UTC() }
		out=append(out,info)
	}
	return out,nil
}

func infoFromInspect(item *ContainerInspect) managed.Info {
	labels:=cloneStringMap(item.Config.Labels)
	return managed.Info{
		ID:item.ID, Name:item.Name, Image:item.Config.Image, State:item.State.Status,
		OwnerID:labels["gomcp.owner_id"], TaskID:labels["gomcp.task_id"],
		ProjectID:labels["gomcp.project_id"], PresetID:labels["gomcp.preset_id"],
		CreatedAt:item.Created, Labels:labels,
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if in==nil{return nil}
	out:=make(map[string]string,len(in))
	for key,value:=range in{out[key]=value}
	return out
}
