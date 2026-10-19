package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"nhooyr.io/websocket"
)

// handleTerminal proxies an interactive terminal between the browser and a container exec session.
func (s *Server) handleTerminal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		s.jsonError(w, "Container ID is required", http.StatusBadRequest)
		return
	}

	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow cross-origin dev if necessary
	})
	if err != nil {
		s.log.Error("websocket accept error", map[string]string{"error": err.Error()})
		return
	}
	defer ws.Close(websocket.StatusInternalError, "closing connection")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// 1. Create exec session in container
	execID, err := s.podman.CreateExec(ctx, id, []string{"/bin/bash"})
	if err != nil {
		// Try fallback to /bin/sh if bash is missing (e.g. Alpine)
		execID, err = s.podman.CreateExec(ctx, id, []string{"/bin/sh"})
		if err != nil {
			_ = ws.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf("\r\nFailed to create exec session: %v\r\n", err)))
			return
		}
	}

	// 2. Connect to Podman socket for raw exec stream
	conn, err := s.podman.DialUnixConn()
	if err != nil {
		_ = ws.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf("\r\nFailed to connect to Podman socket: %v\r\n", err)))
		return
	}
	defer conn.Close()

	// 3. Send HTTP request to start exec with Upgrade / raw hijack
	execStartPayload := `{"Detach":false,"Tty":true}`
	reqHeader := fmt.Sprintf("POST /v4.0.0/libpod/exec/%s/start HTTP/1.1\r\n"+
		"Host: d\r\n"+
		"Content-Type: application/json\r\n"+
		"Connection: Upgrade\r\n"+
		"Upgrade: tcp\r\n"+
		"Content-Length: %d\r\n\r\n%s",
		execID, len(execStartPayload), execStartPayload)

	if _, err := conn.Write([]byte(reqHeader)); err != nil {
		_ = ws.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf("\r\nFailed to send exec start: %v\r\n", err)))
		return
	}

	// Read HTTP response status from connection
	reader := bufio.NewReader(conn)
	statusLine, err := reader.ReadString('\n')
	if err != nil || (!strings.Contains(statusLine, "101") && !strings.Contains(statusLine, "200")) {
		_ = ws.Write(ctx, websocket.MessageText, []byte(fmt.Sprintf("\r\nExec start failed: %s\r\n", statusLine)))
		return
	}

	// Read remaining HTTP headers until empty line
	for {
		line, err := reader.ReadString('\n')
		if err != nil || strings.TrimSpace(line) == "" {
			break
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Pump: Podman -> WebSocket
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := reader.Read(buf)
				if n > 0 {
					wErr := ws.Write(ctx, websocket.MessageBinary, buf[:n])
					if wErr != nil {
						return
					}
				}
				if err != nil {
					return
				}
			}
		}
	}()

	// Pump: WebSocket -> Podman
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				msgType, data, err := ws.Read(ctx)
				if err != nil {
					return
				}
				if msgType == websocket.MessageText || msgType == websocket.MessageBinary {
					// Check if message is a resize control JSON or raw input
					var resizeMsg struct {
						Type string `json:"type"`
						Cols int    `json:"cols"`
						Rows int    `json:"rows"`
					}
					if json.Unmarshal(data, &resizeMsg) == nil && resizeMsg.Type == "resize" {
						// Resize window if supported
						continue
					}

					if _, err := conn.Write(data); err != nil {
						return
					}
				}
			}
		}
	}()

	wg.Wait()
	ws.Close(websocket.StatusNormalClosure, "session ended")
}
