package podman

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	managed "github.com/Mohamed-faaris/go-mcp/internal/container"
)

func TestRuntimeCreateAppliesTaskIsolationAndOwnershipLabels(t *testing.T) {
	labels := map[string]string{
		"gomcp.managed":"true","gomcp.owner_id":"user-1","gomcp.task_id":"task-1",
		"gomcp.project_id":"project-1","gomcp.preset_id":"node-pnpm",
	}
	var createPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
		switch {
		case r.Method==http.MethodPost && r.URL.Path=="/v4.0.0/libpod/containers/create":
			if err:=json.NewDecoder(r.Body).Decode(&createPayload);err!=nil{t.Errorf("decode create body: %v",err)}
			w.WriteHeader(http.StatusCreated)
			_,_=w.Write([]byte(`{"Id":"container-1"}`))
		case r.Method==http.MethodGet && r.URL.Path=="/v4.0.0/libpod/containers/container-1/json":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"Id":"container-1","Name":"gomcp-task-1","Created":"2026-11-10T10:00:00Z",
				"State":map[string]interface{}{"Status":"created"},
				"Config":map[string]interface{}{"Image":"node:22","Labels":labels},
			})
		default:
			http.NotFound(w,r)
		}
	}))
	defer server.Close()

	client:=&Client{httpClient:server.Client(),pullClient:server.Client(),baseURL:server.URL+"/v4.0.0/libpod"}
	runtime,err:=NewRuntime(client);if err!=nil{t.Fatal(err)}
	info,err:=runtime.Create(context.Background(),managed.Spec{
		Name:"gomcp-task-1",Image:"node:22",Workspace:"/srv/gomcp/workspaces/task-1",
		OwnerID:"user-1",TaskID:"task-1",ProjectID:"project-1",PresetID:"node-pnpm",
		Environment:map[string]string{"CI":"true"},Network:"gomcp-agent-egress",
		Limits:managed.ResourceLimits{MemoryBytes:2<<30,CPUCores:2,PidsLimit:512},
		ReadOnlyRoot:true,NoNewPrivileges:true,DropCapabilities:[]string{"ALL"},
	})
	if err!=nil{t.Fatal(err)}
	if info.OwnerID!="user-1"||info.TaskID!="task-1"{t.Fatalf("identity not mapped from inspect labels: %#v",info)}
	if createPayload["read_only_filesystem"]!=true{t.Fatalf("root filesystem is not read-only: %#v",createPayload["read_only_filesystem"])}
	if createPayload["no_new_privileges"]!=true{t.Fatalf("no_new_privileges missing: %#v",createPayload)}
	caps,ok:=createPayload["cap_drop"].([]interface{});if !ok||len(caps)!=1||caps[0]!="ALL"{t.Fatalf("capabilities were not dropped: %#v",createPayload["cap_drop"])}
	resources:=createPayload["resource_limits"].(map[string]interface{})
	pids:=resources["pids"].(map[string]interface{});if pids["limit"]!=float64(512){t.Fatalf("PID limit not applied: %#v",pids)}
	networks:=createPayload["Networks"].(map[string]interface{});if _,ok:=networks["gomcp-agent-egress"];!ok{t.Fatalf("private network not attached: %#v",networks)}
	mounts:=createPayload["mounts"].([]interface{});if len(mounts)!=1{t.Fatalf("expected exactly one workspace mount, got %#v",mounts)}
	mount:=mounts[0].(map[string]interface{});if mount["Source"]!="/srv/gomcp/workspaces/task-1"||mount["Destination"]!="/workspace"{t.Fatalf("unexpected workspace mount: %#v",mount)}
	if _,ok:=createPayload["devices"];ok{t.Fatal("workload adapter must not pass host devices")}
	if _,ok:=createPayload["privileged"];ok{t.Fatal("workload adapter must not request privileged mode")}
}

func TestListContainersDoesNotTrustNamePrefix(t *testing.T) {
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		_ = json.NewEncoder(w).Encode([]ContainerSummary{
			{ID:"lookalike",Names:[]string{"/gomcp-lookalike"},Labels:map[string]string{}},
			{ID:"managed",Names:[]string{"/custom-name"},Labels:map[string]string{"gomcp.managed":"true"}},
		})
	}))
	defer server.Close()
	client:=&Client{httpClient:server.Client(),baseURL:server.URL+"/v4.0.0/libpod"}
	items,err:=client.ListContainers(context.Background());if err!=nil{t.Fatal(err)}
	if len(items)!=1||items[0].ID!="managed"{t.Fatalf("name prefix incorrectly treated as managed: %#v",items)}
}

func TestCreateContainerOverwritesReservedIdentityLabels(t *testing.T) {
	var payload map[string]interface{}
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		_ = json.NewDecoder(r.Body).Decode(&payload)
		w.WriteHeader(http.StatusCreated)
		_,_=w.Write([]byte(`{"Id":"container-2"}`))
	}))
	defer server.Close()
	client:=&Client{httpClient:server.Client(),baseURL:server.URL+"/v4.0.0/libpod"}
	_,err:=client.CreateContainer(context.Background(),CreateContainerOpts{
		Name:"test",Image:"node:22",OwnerID:"trusted-owner",TaskID:"trusted-task",
		ProjectID:"trusted-project",PresetID:"node-pnpm",
		Labels:map[string]string{"gomcp.owner_id":"attacker","gomcp.task_id":"attacker","gomcp.managed":"false"},
	})
	if err!=nil{t.Fatal(err)}
	labels:=payload["labels"].(map[string]interface{})
	if labels["gomcp.managed"]!="true"||labels["gomcp.owner_id"]!="trusted-owner"||labels["gomcp.task_id"]!="trusted-task"||labels["gomcp.project_id"]!="trusted-project"{t.Fatalf("reserved labels were not manager-controlled: %#v",labels)}
}

func TestCreateContainerDropsCallerGomcpNamespaceAndCopiesEnv(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&payload)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"Id":"container-3"}`))
	}))
	defer server.Close()

	client := &Client{httpClient: server.Client(), baseURL: server.URL + "/v4.0.0/libpod"}
	env := make([]string, 1, 8)
	env[0] = "CUSTOM=value"
	_, err := client.CreateContainer(context.Background(), CreateContainerOpts{
		Name: "test", Image: "node:22", OwnerID: "trusted-owner", TaskID: "trusted-task",
		ProjectID: "trusted-project", PresetID: "node-pnpm", Env: env,
		Labels: map[string]string{
			"gomcp.managed": "false",
			"gomcp.future_security_flag": "forged",
			"team": "engineering",
		},
	})
	if err != nil { t.Fatal(err) }
	labels := payload["labels"].(map[string]interface{})
	if labels["gomcp.managed"] != "true" { t.Fatalf("manager managed label missing: %#v", labels) }
	if _, exists := labels["gomcp.future_security_flag"]; exists {
		t.Fatalf("caller controlled gomcp namespace must be dropped: %#v", labels)
	}
	if labels["team"] != "engineering" { t.Fatalf("non-reserved labels should be retained: %#v", labels) }
	if len(env) < 2 || env[1] != "" {
		t.Fatalf("CreateContainer mutated caller-owned env backing array: %#v", env)
	}
}
