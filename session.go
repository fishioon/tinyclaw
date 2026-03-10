package main

import "fmt"

const (
	chatTypeDirect = "direct"
	chatTypeGroup  = "group"
)

type Session struct {
	TenantID   string
	ChatType   string
	SessionKey string
}

func sessionFromMessage(tenantID string, msg *WeComMessage) (Session, error) {
	if msg == nil {
		return Session{}, fmt.Errorf("message is nil")
	}

	sessionKey, err := sessionKey(msg)
	if err != nil {
		return Session{}, err
	}

	return Session{
		TenantID:   tenantID,
		ChatType:   chatType(msg),
		SessionKey: sessionKey,
	}, nil
}

func chatType(msg *WeComMessage) string {
	if msg != nil && msg.RoomID != "" {
		return chatTypeGroup
	}
	return chatTypeDirect
}

func sessionKey(msg *WeComMessage) (string, error) {
	if msg == nil {
		return "", fmt.Errorf("message is nil")
	}
	if msg.RoomID != "" {
		return msg.RoomID, nil
	}
	if msg.From == "" {
		return "", fmt.Errorf("message from is empty")
	}
	return msg.From, nil
}

func streamKey(prefix string, session Session) string {
	return prefix + ":" + session.SessionKey
}
