//go:build integration

package sandbox

import (
	"context"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
)

// Run with: go test ./sandbox/ -tags integration -run TestACL -v
// Requires a real Redis at REDIS_ADDR (default localhost:6379).

func integrationRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("skipping: cannot connect to Redis at %s: %v", addr, err)
	}
	t.Cleanup(func() { rdb.Close() })
	return rdb
}

func TestACL_ProvisionAndScope(t *testing.T) {
	ctx := context.Background()
	rdb := integrationRedis(t)

	orch := &Orchestrator{
		redis: rdb,
		cfg:   Config{},
	}

	roomID := "integration-test-room"
	username := "sb:" + roomID
	inStream := "stream:i:" + roomID
	outStream := "stream:o:" + roomID

	// Cleanup after test
	t.Cleanup(func() {
		rdb.Do(ctx, "ACL", "DELUSER", username)
		rdb.Del(ctx, inStream, outStream)
	})
	// Pre-clean in case previous run left state
	rdb.Do(ctx, "ACL", "DELUSER", username)
	rdb.Del(ctx, inStream, outStream)

	// Provision the user
	cred, err := orch.provisionRedisUser(ctx, roomID)
	if err != nil {
		t.Fatalf("provisionRedisUser: %v", err)
	}
	if cred.Username != username {
		t.Errorf("username = %q, want %q", cred.Username, username)
	}
	if cred.Password == "" {
		t.Fatal("password is empty")
	}

	// Connect as the provisioned user
	userClient := redis.NewClient(&redis.Options{
		Addr:     rdb.Options().Addr,
		Username: cred.Username,
		Password: cred.Password,
	})
	t.Cleanup(func() { userClient.Close() })

	// PING should work
	if err := userClient.Ping(ctx).Err(); err != nil {
		t.Fatalf("user ping failed: %v", err)
	}

	// XGROUP CREATE on own ingress stream should work
	err = userClient.XGroupCreateMkStream(ctx, inStream, "test-group", "0").Err()
	if err != nil {
		t.Fatalf("xgroup create on ingress stream failed: %v", err)
	}

	// XINFO on own ingress stream should work
	err = userClient.Do(ctx, "XINFO", "STREAM", inStream).Err()
	if err != nil {
		t.Fatalf("xinfo on ingress stream failed: %v", err)
	}

	// XACK on own ingress stream should work (no-op, returns 0)
	err = userClient.XAck(ctx, inStream, "test-group", "0-0").Err()
	if err != nil {
		t.Fatalf("xack on ingress stream failed: %v", err)
	}

	// XADD on own egress stream should work
	err = userClient.XAdd(ctx, &redis.XAddArgs{
		Stream: outStream,
		Values: map[string]any{"text": "hello"},
	}).Err()
	if err != nil {
		t.Fatalf("xadd on egress stream failed: %v", err)
	}

	// Access to a different room's stream should be denied
	otherStream := "stream:i:other-room"
	err = userClient.XGroupCreateMkStream(ctx, otherStream, "test-group", "0").Err()
	if err == nil {
		t.Fatal("expected NOPERM error accessing other room's stream, got nil")
	}
	t.Cleanup(func() { rdb.Del(ctx, otherStream) })

	// SET should be denied (not in allowed commands)
	err = userClient.Set(ctx, "some-key", "value", 0).Err()
	if err == nil {
		t.Fatal("expected NOPERM error for SET command, got nil")
	}
}
