package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultSandboxServerPort = 8888

type RouterConfig struct {
	BaseURL    string
	Namespace  string
	ServerPort int
}

type RouterClient struct {
	baseURL    string
	namespace  string
	serverPort int
	httpClient *http.Client
}

type AgentRequest struct {
	Query    string `json:"query"`
	MsgID    string `json:"msgid"`
	RoomID   string `json:"room_id"`
	TenantID string `json:"tenant_id"`
	ChatType string `json:"chat_type"`
}

type ExecutionResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

func NewRouterClient(httpClient *http.Client, cfg RouterConfig) *RouterClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if cfg.ServerPort <= 0 {
		cfg.ServerPort = defaultSandboxServerPort
	}

	return &RouterClient{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		namespace:  cfg.Namespace,
		serverPort: cfg.ServerPort,
		httpClient: httpClient,
	}
}

func (c *RouterClient) Invoke(ctx context.Context, sandboxID string, req AgentRequest) (ExecutionResult, error) {
	if sandboxID == "" {
		return ExecutionResult{}, fmt.Errorf("sandboxID is required")
	}
	if c.baseURL == "" {
		return ExecutionResult{}, fmt.Errorf("router base URL is required")
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("marshal agent request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/agent",
		bytes.NewReader(payload),
	)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("build router request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Sandbox-ID", sandboxID)
	httpReq.Header.Set("X-Sandbox-Namespace", c.namespace)
	httpReq.Header.Set("X-Sandbox-Port", fmt.Sprintf("%d", c.serverPort))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("call sandbox router: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("read sandbox response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return ExecutionResult{}, fmt.Errorf("sandbox router returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result ExecutionResult
	if err := json.Unmarshal(body, &result); err != nil {
		return ExecutionResult{}, fmt.Errorf("decode sandbox response: %w", err)
	}
	if result.ExitCode != 0 {
		errText := strings.TrimSpace(result.Stderr)
		if errText == "" {
			errText = strings.TrimSpace(result.Stdout)
		}
		if errText == "" {
			errText = "unknown agent runtime failure"
		}
		return ExecutionResult{}, fmt.Errorf("sandbox agent failed with exit_code=%d: %s", result.ExitCode, errText)
	}
	if strings.TrimSpace(result.Stdout) == "" {
		return ExecutionResult{}, fmt.Errorf("sandbox response stdout is empty")
	}

	return result, nil
}
