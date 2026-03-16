package sandbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterClientInvoke(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/agent" {
			t.Fatalf("path = %s, want /agent", r.URL.Path)
		}
		if got := r.Header.Get("X-Sandbox-ID"); got != "clawagent-room-1" {
			t.Fatalf("X-Sandbox-ID = %q, want %q", got, "clawagent-room-1")
		}
		if got := r.Header.Get("X-Sandbox-Namespace"); got != "claw" {
			t.Fatalf("X-Sandbox-Namespace = %q, want %q", got, "claw")
		}
		if got := r.Header.Get("X-Sandbox-Port"); got != "8888" {
			t.Fatalf("X-Sandbox-Port = %q, want %q", got, "8888")
		}

		var req AgentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Query != "hello" {
			t.Fatalf("request query = %q, want %q", req.Query, "hello")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ExecutionResult{Stdout: "sandbox reply", ExitCode: 0})
	}))
	defer server.Close()

	client := NewRouterClient(server.Client(), RouterConfig{
		BaseURL:    server.URL,
		Namespace:  "claw",
		ServerPort: 8888,
	})

	resp, err := client.Invoke(context.Background(), "clawagent-room-1", AgentRequest{
		Query:    "hello",
		MsgID:    "msg-1",
		RoomID:   "room-1",
		TenantID: "corp-id",
		ChatType: "group",
	})
	if err != nil {
		t.Fatalf("Invoke returned error: %v", err)
	}
	if resp.Stdout != "sandbox reply" {
		t.Fatalf("response stdout = %q, want %q", resp.Stdout, "sandbox reply")
	}
}

func TestRouterClientInvoke_PropagatesHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "sandbox failed", http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewRouterClient(server.Client(), RouterConfig{
		BaseURL:    server.URL,
		Namespace:  "claw",
		ServerPort: 8888,
	})

	_, err := client.Invoke(context.Background(), "clawagent-room-1", AgentRequest{
		Query:    "hello",
		MsgID:    "msg-1",
		RoomID:   "room-1",
		TenantID: "corp-id",
		ChatType: "group",
	})
	if err == nil {
		t.Fatal("Invoke error = nil, want non-nil")
	}
}

func TestRouterClientInvoke_PropagatesAgentFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ExecutionResult{
			Stdout:   "",
			Stderr:   "agent failed",
			ExitCode: 1,
		})
	}))
	defer server.Close()

	client := NewRouterClient(server.Client(), RouterConfig{
		BaseURL:    server.URL,
		Namespace:  "claw",
		ServerPort: 8888,
	})

	_, err := client.Invoke(context.Background(), "clawagent-room-1", AgentRequest{Query: "hello"})
	if err == nil {
		t.Fatal("Invoke error = nil, want non-nil")
	}
}
