package podman

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStopContainerWithTimeoutSendsPodmanTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v4.0.0/libpod/containers/container-1/stop" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if got := r.URL.Query().Get("t"); got != "3" {
			t.Fatalf("expected stop timeout t=3, got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := &Client{httpClient: server.Client(), baseURL: server.URL + "/v4.0.0/libpod"}
	if err := client.StopContainerWithTimeout(context.Background(), "container-1", 2500*time.Millisecond); err != nil {
		t.Fatal(err)
	}
}

func TestStopContainerUsesDefaultTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("t"); got != "" {
			t.Fatalf("expected default timeout parameter to be omitted, got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := &Client{httpClient: server.Client(), baseURL: server.URL + "/v4.0.0/libpod"}
	if err := client.StopContainer(context.Background(), "container-1"); err != nil {
		t.Fatal(err)
	}
}
