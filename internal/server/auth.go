package server

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
)

type contextKey string

const (
	roleContextKey     contextKey = "user_role"
	usernameContextKey contextKey = "user_name"
)

// withAuth enforces HTTP Basic Authentication and role requirements ("viewer" or "admin").
func (s *Server) withAuth(next http.HandlerFunc, requiredRole string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If auth mode is disabled, grant admin access directly
		if strings.EqualFold(s.cfg.Auth.Mode, "none") {
			ctx := context.WithValue(r.Context(), roleContextKey, "admin")
			ctx = context.WithValue(ctx, usernameContextKey, "anonymous")
			next(w, r.WithContext(ctx))
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Also check query param ?auth= for WebSockets if needed
			if qAuth := r.URL.Query().Get("token"); qAuth != "" {
				authHeader = "Basic " + qAuth
			}
		}

		if !strings.HasPrefix(authHeader, "Basic ") {
			w.Header().Set("WWW-Authenticate", `Basic realm="go-mcp"`)
			s.jsonError(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
		if err != nil {
			s.jsonError(w, "Invalid authorization header encoding", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(string(payload), ":", 2)
		if len(parts) != 2 {
			s.jsonError(w, "Invalid credentials format", http.StatusUnauthorized)
			return
		}

		username, password := parts[0], parts[1]
		var userRole string

		if username == s.cfg.Auth.Admin.Username && password == s.cfg.Auth.Admin.Password {
			userRole = "admin"
		} else if username == s.cfg.Auth.Viewer.Username && password == s.cfg.Auth.Viewer.Password {
			userRole = "viewer"
		} else {
			s.jsonError(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Check role permissions: Admin satisfies both admin and viewer
		if requiredRole == "admin" && userRole != "admin" {
			s.jsonError(w, "Forbidden: Admin privileges required", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), roleContextKey, userRole)
		ctx = context.WithValue(ctx, usernameContextKey, username)
		next(w, r.WithContext(ctx))
	}
}

func getUserRole(ctx context.Context) string {
	if r, ok := ctx.Value(roleContextKey).(string); ok {
		return r
	}
	return "viewer"
}
