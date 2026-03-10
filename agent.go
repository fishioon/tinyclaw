package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	consumerGroup = "tinyclaw"
	consumerName  = "agent"
	blockTimeout  = 5 * time.Second
)

// Agent consumes messages from a Redis stream and processes them via Claude agent SDK
type Agent struct {
	cfg    Config
	redis  *redis.Client
	stream string
}

func NewAgent(cfg Config, rdb *redis.Client, stream string) *Agent {
	return &Agent{cfg: cfg, redis: rdb, stream: stream}
}

// Run starts the agent consumer loop
func (a *Agent) Run(ctx context.Context) error {
	// Ensure consumer group exists
	err := a.redis.XGroupCreateMkStream(ctx, a.stream, consumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create consumer group: %w", err)
	}

	log.Printf("agent started stream=%s group=%s", a.stream, consumerGroup)

	for {
		if ctx.Err() != nil {
			return nil
		}

		msgs, err := a.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: consumerName,
			Streams:  []string{a.stream, ">"},
			Count:    1,
			Block:    blockTimeout,
		}).Result()

		if err != nil {
			if err == redis.Nil || strings.Contains(err.Error(), "context") {
				continue
			}
			log.Printf("xreadgroup error: %v", err)
			continue
		}

		for _, stream := range msgs {
			for _, msg := range stream.Messages {
				if err := a.process(ctx, msg); err != nil {
					log.Printf("process error msgid=%s err=%v", msg.ID, err)
					continue
				}
				// XACK only after successful processing
				if err := a.redis.XAck(ctx, a.stream, consumerGroup, msg.ID).Err(); err != nil {
					log.Printf("xack error msgid=%s err=%v", msg.ID, err)
				}
			}
		}
	}
}

// process passes the message to Claude agent SDK and handles the response
func (a *Agent) process(ctx context.Context, msg redis.XMessage) error {
	raw, ok := msg.Values["raw"].(string)
	if !ok || raw == "" {
		return fmt.Errorf("missing raw field in message")
	}

	// Invoke Claude agent SDK - agent has full autonomy over how to respond
	output, err := runClaudeAgent(ctx, raw)
	if err != nil {
		return fmt.Errorf("claude agent: %w", err)
	}

	log.Printf("agent processed msgid=%s output_len=%d", msg.ID, len(output))
	return nil
}

// runClaudeAgent invokes the Claude agent SDK with the message content
// The agent decides autonomously how to respond
func runClaudeAgent(ctx context.Context, input string) (string, error) {
	cmd := exec.CommandContext(ctx, "claude", "-p", input, "--output-format", "text")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("claude exec: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
