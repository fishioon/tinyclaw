package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestRouterHealthz(t *testing.T) {
	handler := newRouterHandler(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			t.Fatalf("unexpected proxy request: %s", req.URL.String())
			return nil, nil
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("body = %q, want status ok", rec.Body.String())
	}
}

func TestRouterRejectsMissingSandboxID(t *testing.T) {
	handler := newRouterHandler(&http.Client{})

	req := httptest.NewRequest(http.MethodPost, "/agent", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestRouterProxiesRequest(t *testing.T) {
	var gotURL string
	var gotHost string
	var gotHeader string
	var gotBody string

	handler := newRouterHandler(&http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			gotURL = req.URL.String()
			gotHost = req.Host
			gotHeader = req.Header.Get("X-Test")
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			gotBody = string(body)

			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body: io.NopCloser(strings.NewReader(`{"stdout":"ok","stderr":"","exit_code":0}`)),
			}, nil
		}),
	})

	req := httptest.NewRequest(http.MethodPost, "/agent?trace=1", strings.NewReader(`{"query":"hello"}`))
	req.Host = "sandbox-router-svc.claw.svc.cluster.local"
	req.Header.Set("X-Sandbox-ID", "clawagent-room-1")
	req.Header.Set("X-Sandbox-Namespace", "claw")
	req.Header.Set("X-Sandbox-Port", "8888")
	req.Header.Set("X-Test", "forwarded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotURL != "http://clawagent-room-1.claw.svc.cluster.local:8888/agent?trace=1" {
		t.Fatalf("proxy url = %q", gotURL)
	}
	if gotHost != "" {
		t.Fatalf("proxy host = %q, want empty", gotHost)
	}
	if gotHeader != "forwarded" {
		t.Fatalf("X-Test = %q, want forwarded", gotHeader)
	}
	if gotBody != `{"query":"hello"}` {
		t.Fatalf("body = %q", gotBody)
	}
}
