package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"k8s.io/client-go/rest"
	sandboxclient "sigs.k8s.io/agent-sandbox/clients/k8s/clientset/versioned"
	"tinyclaw/sandbox"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		cancel()
		log.Fatalf("redis ping failed: %v", err)
	}
	cancel()

	var orch *sandbox.Orchestrator
	if cfg.SandboxEnabled {
		k8sCfg, err := rest.InClusterConfig()
		if err != nil {
			log.Fatalf("k8s in-cluster config: %v", err)
		}
		clientset, err := sandboxclient.NewForConfig(k8sCfg)
		if err != nil {
			log.Fatalf("k8s clientset: %v", err)
		}
		orch = sandbox.NewOrchestrator(clientset, redisClient, sandbox.Config{
			Namespace:    cfg.SandboxNamespace,
			Image:        cfg.SandboxImage,
			RedisAddr:    cfg.RedisAddr,
			StreamPrefix: cfg.StreamPrefix,
		})
		log.Printf("sandbox orchestrator enabled: namespace=%s image=%s", cfg.SandboxNamespace, cfg.SandboxImage)
	}

	clawman, err := NewClawman(cfg, redisClient, orch)
	if err != nil {
		log.Fatalf("init clawman: %v", err)
	}
	defer clawman.Close()

	runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := clawman.Run(runCtx); err != nil {
		log.Fatalf("clawman stopped with error: %v", err)
	}
	log.Printf("clawman stopped")
}
