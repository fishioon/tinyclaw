package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// 集成测试：验证 spawn → ensure → announce 完整流程
func TestSpawnIntegration(t *testing.T) {
	// 使用 fake 组件模拟完整流程
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rdb.Close()

	ctx := context.Background()
	
	// 清理测试数据
	testParentKey := "test-parent-" + newID()[:8]
	defer func() {
		keys, _ := rdb.Keys(ctx, "stream:session:"+testParentKey+"*").Result()
		if len(keys) > 0 {
			rdb.Del(ctx, keys...)
		}
	}()

	fakeEnsurer := &fakeSpawnEnsurer{}
	spawner := NewSessionSpawner(rdb, fakeEnsurer, "stream:session")

	// 1. 父 agent 调用 Spawn
	resp, err := spawner.Spawn(ctx, SpawnRequest{
		ParentSessionKey: testParentKey,
		AgentID:          "sql-agent",
		Task:             "查询本月销售总额",
		TenantID:         "test-tenant",
		ChatType:         "group",
	})

	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}
	if resp.ChildSessionKey == "" {
		t.Fatal("ChildSessionKey is empty")
	}
	if resp.AgentID != "sql-agent" {
		t.Fatalf("AgentID = %q, want %q", resp.AgentID, "sql-agent")
	}

	// 2. 验证子 stream 已创建
	childStreamKey := "stream:session:" + resp.ChildSessionKey
	entries, err := rdb.XRange(ctx, childStreamKey, "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("child stream entries = %d, want 1", len(entries))
	}

	// 3. 验证 ensure 被调用
	if fakeEnsurer.calls != 1 {
		t.Fatalf("ensure calls = %d, want 1", fakeEnsurer.calls)
	}
	if fakeEnsurer.lastSessionKey != resp.ChildSessionKey {
		t.Fatalf("ensure session_key = %q, want %q", fakeEnsurer.lastSessionKey, resp.ChildSessionKey)
	}

	// 4. 模拟子 agent 完成任务并 announce
	err = spawner.Announce(ctx, AnnounceRequest{
		ParentSessionKey: testParentKey,
		ChildSessionKey:  resp.ChildSessionKey,
		AgentID:          "sql-agent",
		Result:           "本月销售总额: ¥1,234,567",
		Status:           "success",
	})
	if err != nil {
		t.Fatalf("Announce() error = %v", err)
	}

	// 5. 验证父 stream 收到结果
	parentStreamKey := "stream:session:" + testParentKey
	parentEntries, err := rdb.XRange(ctx, parentStreamKey, "-", "+").Result()
	if err != nil {
		t.Fatalf("XRange() error = %v", err)
	}
	if len(parentEntries) != 1 {
		t.Fatalf("parent stream entries = %d, want 1", len(parentEntries))
	}

	// 6. 验证结果内容
	resultEvent := parentEntries[0].Values
	if resultEvent["event_type"] != eventTypeSubagentResult {
		t.Fatalf("event_type = %q, want %q", resultEvent["event_type"], eventTypeSubagentResult)
	}
	if resultEvent["content"] != "本月销售总额: ¥1,234,567" {
		t.Fatalf("content = %q, want %q", resultEvent["content"], "本月销售总额: ¥1,234,567")
	}

	var metadata map[string]interface{}
	json.Unmarshal([]byte(resultEvent["metadata"].(string)), &metadata)
	if metadata["child_session_key"] != resp.ChildSessionKey {
		t.Fatalf("metadata.child_session_key = %q, want %q", metadata["child_session_key"], resp.ChildSessionKey)
	}
}

func TestSpawnValidation(t *testing.T) {
	spawner := NewSessionSpawner(nil, nil, "stream:session")
	ctx := context.Background()

	tests := []struct {
		name    string
		req     SpawnRequest
		wantErr string
	}{
		{
			name:    "missing parent_session_key",
			req:     SpawnRequest{AgentID: "sql-agent", Task: "test"},
			wantErr: "parent_session_key is required",
		},
		{
			name:    "missing agent_id",
			req:     SpawnRequest{ParentSessionKey: "parent-1", Task: "test"},
			wantErr: "agent_id is required",
		},
		{
			name:    "missing task",
			req:     SpawnRequest{ParentSessionKey: "parent-1", AgentID: "sql-agent"},
			wantErr: "task is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := spawner.Spawn(ctx, tt.req)
			if err == nil {
				t.Fatal("Spawn() error = nil, want non-nil")
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("Spawn() error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestChildSessionKeyFormat(t *testing.T) {
	parent := "room-123"
	child := childSessionKey(parent)
	
	if len(child) != len(parent)+len(":subagent:")+16 {
		t.Fatalf("child key length = %d, want %d", len(child), len(parent)+len(":subagent:")+16)
	}
	if child[:len(parent)] != parent {
		t.Fatalf("child key prefix = %q, want %q", child[:len(parent)], parent)
	}
	if child[len(parent):len(parent)+10] != ":subagent:" {
		t.Fatalf("child key separator = %q, want %q", child[len(parent):len(parent)+10], ":subagent:")
	}
}

// fakeSpawnEnsurer 用于测试的 fake ensurer
type fakeSpawnEnsurer struct {
	calls          int
	lastSessionKey string
	err            error
}

func (f *fakeSpawnEnsurer) Ensure(ctx context.Context, event IngressEvent) error {
	f.calls++
	f.lastSessionKey = event.SessionKey
	return f.err
}
