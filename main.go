package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
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

	clawman, err := NewClawman(cfg, redisClient)
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
