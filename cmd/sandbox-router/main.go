package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultNamespace    = "default"
	defaultSandboxPort  = 8888
	defaultListenAddr   = ":8080"
	defaultProxyTimeout = 180 * time.Second
)

func main() {
	timeout := loadProxyTimeout()
	client := &http.Client{
		Timeout: timeout,
	}

	server := &http.Server{
		Addr:    envOrDefault("LISTEN_ADDR", defaultListenAddr),
		Handler: newRouterHandler(client),
	}

	slog.Info("sandbox router starting", "addr", server.Addr, "proxy_timeout", timeout.String())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("sandbox router stopped", "err", err)
		os.Exit(1)
	}
}

func newRouterHandler(client *http.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/healthz" {
			writeJSON(w, http.StatusOK, `{"status":"ok"}`)
			return
		}

		sandboxID := strings.TrimSpace(r.Header.Get("X-Sandbox-ID"))
		if sandboxID == "" {
			http.Error(w, "X-Sandbox-ID header is required", http.StatusBadRequest)
			return
		}

		namespace := r.Header.Get("X-Sandbox-Namespace")
		if namespace == "" {
			namespace = defaultNamespace
		}
		if !isValidNamespace(namespace) {
			http.Error(w, "invalid namespace format", http.StatusBadRequest)
			return
		}

		port := defaultSandboxPort
		if raw := strings.TrimSpace(r.Header.Get("X-Sandbox-Port")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed <= 0 {
				http.Error(w, "invalid port format", http.StatusBadRequest)
				return
			}
			port = parsed
		}

		targetURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d%s", sandboxID, namespace, port, r.URL.RequestURI())
		req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
		if err != nil {
			http.Error(w, "build proxy request failed", http.StatusInternalServerError)
			return
		}
		req.Header = cloneHeaders(r.Header)
		req.Host = ""
		req.RequestURI = ""

		resp, err := client.Do(req)
		if err != nil {
			slog.Error("proxy request failed", "sandbox_id", sandboxID, "target_url", targetURL, "err", err)
			http.Error(w, "could not connect to backend sandbox", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		copyHeaders(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		if _, err := io.Copy(w, resp.Body); err != nil && !isCanceled(r.Context()) {
			slog.Error("proxy response copy failed", "sandbox_id", sandboxID, "target_url", targetURL, "err", err)
		}
	})
}

func loadProxyTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("PROXY_TIMEOUT_SECONDS"))
	if raw == "" {
		return defaultProxyTimeout
	}

	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil || seconds <= 0 {
		return defaultProxyTimeout
	}
	return time.Duration(seconds * float64(time.Second))
}

func isValidNamespace(namespace string) bool {
	for _, ch := range namespace {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
			continue
		}
		return false
	}
	return namespace != ""
}

func cloneHeaders(src http.Header) http.Header {
	dst := make(http.Header, len(src))
	for key, values := range src {
		if strings.EqualFold(key, "Host") {
			continue
		}
		copied := make([]string, len(values))
		copy(copied, values)
		dst[key] = copied
	}
	return dst
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func writeJSON(w http.ResponseWriter, statusCode int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = io.WriteString(w, body)
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func isCanceled(ctx context.Context) bool {
	return ctx.Err() != nil
}
