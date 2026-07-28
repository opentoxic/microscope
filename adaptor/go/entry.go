package microscope

import (
	"encoding/json"
	"time"
)

// EntryType identifies the kind of recorded activity.
type EntryType string

const (
	TypeRequest      EntryType = "request"
	TypeQuery        EntryType = "query"
	TypeLog          EntryType = "log"
	TypeEvent        EntryType = "event"
	TypeNotification EntryType = "notification"
	TypeException    EntryType = "exception"
	TypeCache        EntryType = "cache"
	TypeRedis        EntryType = "redis"
	TypeJob          EntryType = "job"
	TypeSchedule     EntryType = "schedule"
	TypeMail         EntryType = "mail"
	TypeHTTPClient   EntryType = "http-client"
	TypeWebSocket    EntryType = "websocket"
	TypePerformance  EntryType = "performance"
	TypeMetric       EntryType = "metric"
	TypeCustom       EntryType = "custom"
	TypeTopic        EntryType = "topic"
)

// AllEntryTypes is the complete set of configurable observation signals.
var AllEntryTypes = []EntryType{
	TypeRequest,
	TypeQuery,
	TypeLog,
	TypeEvent,
	TypeNotification,
	TypeException,
	TypeCache,
	TypeRedis,
	TypeJob,
	TypeSchedule,
	TypeMail,
	TypeHTTPClient,
	TypeWebSocket,
	TypePerformance,
	TypeMetric,
	TypeCustom,
	TypeTopic,
}

// ValidEntryType reports whether a type is a supported observation signal.
func ValidEntryType(entryType EntryType) bool {
	for _, candidate := range AllEntryTypes {
		if candidate == entryType {
			return true
		}
	}
	return false
}

// Entry is a single microscope observation.
type Entry struct {
	ID            string         `json:"id"`
	BatchID       string         `json:"batch_id"`
	Type          EntryType      `json:"type"`
	RequestID     string         `json:"request_id,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Tags          []string       `json:"tags,omitempty"`
	Content       map[string]any `json:"content"`
	CreatedAt     time.Time      `json:"created_at"`
}

// ListFilter narrows entry queries.
type ListFilter struct {
	Type      EntryType
	RequestID string
	Search    string
	Limit     int
	Offset    int
}

// ListResult is a paginated list of entries.
type ListResult struct {
	Entries []Entry `json:"entries"`
	Total   int     `json:"total"`
}

func encodeTags(tags []string) ([]byte, error) {
	if tags == nil {
		tags = []string{}
	}
	return json.Marshal(tags)
}

func encodeContent(content map[string]any) ([]byte, error) {
	if content == nil {
		content = map[string]any{}
	}
	return json.Marshal(content)
}
