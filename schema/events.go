package schema

// EventType defines the type of event in the session stream
type EventType string

const (
	EventTypeMessage EventType = "message" // incoming user message
	EventTypeReply   EventType = "reply"   // agent reply to user
	EventTypeError   EventType = "error"   // error event
)

// Event is the unified event schema for session streams
type Event struct {
	ID        string            `json:"id"`         // Redis stream entry ID
	Type      EventType         `json:"type"`       // event type
	SessionID string            `json:"session_id"` // session key (chat_id or user_id)
	Payload   string            `json:"payload"`    // message content
	Meta      map[string]string `json:"meta"`       // optional metadata (source, user_id, etc.)
	CreatedAt int64             `json:"created_at"` // unix timestamp ms
}

// Redis key helpers
const (
	// StreamKey returns the Redis stream key for a session
	// Format: stream:session:{session_key}
	StreamKeyPrefix = "stream:session:"

	// ConsumerGroupPrefix is the consumer group prefix
	// Format: cg:session:{session_key}
	ConsumerGroupPrefix = "cg:session:"

	// EnsureLockPrefix is the ensure lock key prefix (SET NX EX 3)
	// Format: lock:ensure:{session_key}
	EnsureLockPrefix = "lock:ensure:"

	// DLQReply is the dead letter queue for failed replies
	DLQReply = "dlq:reply"

	// DLQRuntime is the dead letter queue for sandbox failures
	DLQRuntime = "runtime_dlq"
)

func StreamKey(sessionKey string) string {
	return StreamKeyPrefix + sessionKey
}

func ConsumerGroup(sessionKey string) string {
	return ConsumerGroupPrefix + sessionKey
}

func EnsureLock(sessionKey string) string {
	return EnsureLockPrefix + sessionKey
}
