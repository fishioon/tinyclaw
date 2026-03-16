package main

import (
	"fmt"
	"os"
	"strconv"
)

const (
	defaultRedisAddr        = "127.0.0.1:6379"
	defaultWeComSeqKey      = "msg:seq"
	defaultSandboxNamespace = "claw"
	defaultSandboxTemplate  = "tinyclaw-agent-template"
)

type Config struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	WeComCorpID        string
	WeComCorpSecret    string
	WeComPrivateKey    string
	WeComSeqKey        string
	WeComContactSecret string
	WeComBotID         string

	SandboxNamespace       string
	SandboxTemplateName    string
	SandboxRouterURL       string
	SandboxServerPort      int
	SandboxReadyTimeoutSec int

	WorkToolRobotID string
}

func LoadConfig() (Config, error) {
	redisDB := parseIntEnv("REDIS_DB", 0)
	sandboxNamespace := envOrDefault("SANDBOX_NAMESPACE", defaultSandboxNamespace)
	sandboxRouterURL := os.Getenv("SANDBOX_ROUTER_URL")
	if sandboxRouterURL == "" {
		sandboxRouterURL = fmt.Sprintf("http://sandbox-router-svc.%s.svc.cluster.local:8080", sandboxNamespace)
	}
	cfg := Config{
		RedisAddr:     envOrDefault("REDIS_ADDR", defaultRedisAddr),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       redisDB,

		WeComCorpID:        os.Getenv("WECOM_CORP_ID"),
		WeComCorpSecret:    os.Getenv("WECOM_CORP_SECRET"),
		WeComPrivateKey:    os.Getenv("WECOM_RSA_PRIVATE_KEY"),
		WeComSeqKey:        envOrDefault("WECOM_SEQ_KEY", defaultWeComSeqKey),
		WeComContactSecret: os.Getenv("WECOM_CONTACT_SECRET"),
		WeComBotID:         os.Getenv("WECOM_BOT_ID"),

		SandboxNamespace:       sandboxNamespace,
		SandboxTemplateName:    envOrDefault("SANDBOX_TEMPLATE_NAME", defaultSandboxTemplate),
		SandboxRouterURL:       sandboxRouterURL,
		SandboxServerPort:      parseIntEnv("SANDBOX_SERVER_PORT", 8888),
		SandboxReadyTimeoutSec: parseIntEnv("SANDBOX_READY_TIMEOUT_SEC", 180),

		WorkToolRobotID: os.Getenv("WORKTOOL_ROBOT_ID"),
	}

	return cfg, nil
}

func parseIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
