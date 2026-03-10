package main

import (
	"testing"
	"time"
)

func TestBuildIngressEventForGroupMessage(t *testing.T) {
	msg := &WeComMessage{
		MsgID:      "msg-1",
		From:       "alice",
		ToList:     []string{"bot"},
		RoomID:     "room-123",
		MsgTime:    1_741_564_800,
		MsgType:    "text",
		RawContent: `{"msgid":"msg-1"}`,
	}

	event, err := buildIngressEvent("corp-id", msg)
	if err != nil {
		t.Fatalf("buildIngressEvent() error = %v", err)
	}
	if event.EventID != "wecom:msg-1" {
		t.Fatalf("EventID = %q, want %q", event.EventID, "wecom:msg-1")
	}
	if event.EventType != eventTypeMessageReceived {
		t.Fatalf("EventType = %q, want %q", event.EventType, eventTypeMessageReceived)
	}
	if event.SessionKey != "room-123" {
		t.Fatalf("SessionKey = %q, want %q", event.SessionKey, "room-123")
	}
	if event.ChatType != chatTypeGroup {
		t.Fatalf("ChatType = %q, want %q", event.ChatType, chatTypeGroup)
	}
	if event.ContentType != "text" {
		t.Fatalf("ContentType = %q, want %q", event.ContentType, "text")
	}
	if event.TraceID != event.EventID {
		t.Fatalf("TraceID = %q, want %q", event.TraceID, event.EventID)
	}
	if event.OccurredAt.Format(time.RFC3339) != "2025-03-10T00:00:00Z" {
		t.Fatalf("OccurredAt = %q, want %q", event.OccurredAt.Format(time.RFC3339), "2025-03-10T00:00:00Z")
	}
}

func TestBuildIngressEventForDirectMessageUsesSenderAsSessionKey(t *testing.T) {
	msg := &WeComMessage{
		MsgID:      "msg-2",
		From:       "employee-a",
		ToList:     []string{"tinyclaw-bot"},
		MsgTime:    1_741_564_800_123,
		MsgType:    "file",
		RawContent: `{"msgid":"msg-2"}`,
	}

	event, err := buildIngressEvent("corp-id", msg)
	if err != nil {
		t.Fatalf("buildIngressEvent() error = %v", err)
	}
	if event.SessionKey != "employee-a" {
		t.Fatalf("SessionKey = %q, want %q", event.SessionKey, "employee-a")
	}
	if event.ChatType != chatTypeDirect {
		t.Fatalf("ChatType = %q, want %q", event.ChatType, chatTypeDirect)
	}
	if event.ContentType != "file" {
		t.Fatalf("ContentType = %q, want %q", event.ContentType, "file")
	}
	if event.OccurredAt.Format(time.RFC3339Nano) != "2025-03-10T00:00:00.123Z" {
		t.Fatalf("OccurredAt = %q, want %q", event.OccurredAt.Format(time.RFC3339Nano), "2025-03-10T00:00:00.123Z")
	}
}

func TestStreamValuesUsesEventSchema(t *testing.T) {
	event := IngressEvent{
		EventID:     "wecom:msg-1",
		EventType:   eventTypeMessageReceived,
		TenantID:    "corp-id",
		SessionKey:  "employee-a",
		SourceMsgID: "msg-1",
		SenderID:    "employee-a",
		ChatType:    chatTypeDirect,
		ContentType: "text",
		Content:     `{"msgid":"msg-1"}`,
		Attachments: "[]",
		OccurredAt:  time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC),
		TraceID:     "wecom:msg-1",
		Raw:         `{"msgid":"msg-1"}`,
	}

	values := streamValues(event)
	if got := values["event_id"]; got != event.EventID {
		t.Fatalf("streamValues()[event_id] = %v, want %q", got, event.EventID)
	}
	if got := values["event_type"]; got != event.EventType {
		t.Fatalf("streamValues()[event_type] = %v, want %q", got, event.EventType)
	}
	if got := values["session_key"]; got != event.SessionKey {
		t.Fatalf("streamValues()[session_key] = %v, want %q", got, event.SessionKey)
	}
	if got := values["occurred_at"]; got != "2025-03-10T00:00:00Z" {
		t.Fatalf("streamValues()[occurred_at] = %v, want %q", got, "2025-03-10T00:00:00Z")
	}
	if got := values["msgid"]; got != event.SourceMsgID {
		t.Fatalf("streamValues()[msgid] = %v, want %q", got, event.SourceMsgID)
	}
	if got := values["raw"]; got != event.Raw {
		t.Fatalf("streamValues()[raw] = %v, want %q", got, event.Raw)
	}
	if got := values["attachments"]; got != "[]" {
		t.Fatalf("streamValues()[attachments] = %v, want %q", got, "[]")
	}
}

func TestNormalizeContentTypeFallbacksToMixed(t *testing.T) {
	if got := normalizeContentType("unknown"); got != "mixed" {
		t.Fatalf("normalizeContentType() = %q, want %q", got, "mixed")
	}
}
