package main

import "testing"

func TestStreamKeyUsesSessionKey(t *testing.T) {
	session := Session{SessionKey: "room-123"}

	if got := streamKey("stream:session", session); got != "stream:session:room-123" {
		t.Fatalf("streamKey() = %q, want %q", got, "stream:session:room-123")
	}
}

func TestSessionFromMessageUsesRoomIDForGroupChat(t *testing.T) {
	msg := &WeComMessage{
		From:   "alice",
		ToList: []string{"bot"},
		RoomID: "room-123",
	}

	session, err := sessionFromMessage("corp-id", msg)
	if err != nil {
		t.Fatalf("sessionFromMessage() error = %v", err)
	}
	if session.SessionKey != "room-123" {
		t.Fatalf("SessionKey = %q, want %q", session.SessionKey, "room-123")
	}
	if session.ChatType != chatTypeGroup {
		t.Fatalf("ChatType = %q, want %q", session.ChatType, chatTypeGroup)
	}
	if session.TenantID != "corp-id" {
		t.Fatalf("TenantID = %q, want %q", session.TenantID, "corp-id")
	}
}

func TestSessionFromMessageSortsDirectParticipants(t *testing.T) {
	msg := &WeComMessage{
		From:   "z-user",
		ToList: []string{"a-user"},
	}

	session, err := sessionFromMessage("corp-id", msg)
	if err != nil {
		t.Fatalf("sessionFromMessage() error = %v", err)
	}
	if session.SessionKey != "z-user" {
		t.Fatalf("SessionKey = %q, want %q", session.SessionKey, "z-user")
	}
	if session.ChatType != chatTypeDirect {
		t.Fatalf("ChatType = %q, want %q", session.ChatType, chatTypeDirect)
	}
}

func TestNewClawmanRequiresWeComConfig(t *testing.T) {
	_, err := NewClawman(Config{}, nil)
	if err == nil {
		t.Fatal("NewClawman() error = nil, want required config error")
	}

	want := "WECOM_CORP_ID/WECOM_CORP_SECRET/WECOM_RSA_PRIVATE_KEY are required"
	if err.Error() != want {
		t.Fatalf("NewClawman() error = %q, want %q", err.Error(), want)
	}
}

func TestSessionFromMessageRejectsMissingDirectParticipants(t *testing.T) {
	_, err := sessionFromMessage("corp-id", &WeComMessage{From: "alice"})
	if err != nil {
		t.Fatalf("sessionFromMessage() error = %v, want nil", err)
	}
}
