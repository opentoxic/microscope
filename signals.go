package microscope

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// RecordCache records a cache hit, miss, write, forget, or flush operation.
func (h *Hub) RecordCache(ctx context.Context, operation, key string, hit bool, duration time.Duration, content map[string]any) {
	payload := cloneContent(content)
	payload["operation"] = operation
	payload["key"] = key
	payload["hit"] = hit
	payload["duration_ms"] = milliseconds(duration)
	h.recordTyped(ctx, TypeCache, []string{"cache:" + operation}, payload)
}

// RecordRedis records a Redis command or pipeline execution.
func (h *Hub) RecordRedis(ctx context.Context, command string, duration time.Duration, err error, content map[string]any) {
	payload := cloneContent(content)
	payload["command"] = command
	payload["duration_ms"] = milliseconds(duration)
	if err != nil {
		payload["error"] = err.Error()
	}
	h.recordTyped(ctx, TypeRedis, []string{"redis:" + command}, payload)
}

// RecordJob records a queued job lifecycle transition.
func (h *Hub) RecordJob(ctx context.Context, name, queue, state string, duration time.Duration, content map[string]any) {
	payload := cloneContent(content)
	payload["name"] = name
	payload["queue"] = queue
	payload["state"] = state
	payload["duration_ms"] = milliseconds(duration)
	h.recordTyped(ctx, TypeJob, []string{"job:" + state, "queue:" + queue}, payload)
}

// RecordSchedule records a scheduled task invocation.
func (h *Hub) RecordSchedule(ctx context.Context, name, state string, duration time.Duration, content map[string]any) {
	payload := cloneContent(content)
	payload["name"] = name
	payload["state"] = state
	payload["duration_ms"] = milliseconds(duration)
	h.recordTyped(ctx, TypeSchedule, []string{"schedule:" + state}, payload)
}

// RecordMail records a mail delivery attempt without storing the message body.
func (h *Hub) RecordMail(ctx context.Context, subject string, recipients []string, state string, duration time.Duration, content map[string]any) {
	payload := cloneContent(content)
	payload["subject"] = subject
	payload["recipients"] = recipients
	payload["state"] = state
	payload["duration_ms"] = milliseconds(duration)
	h.recordTyped(ctx, TypeMail, []string{"mail:" + state}, payload)
}

// RecordWebSocket records a WebSocket connection, message, broadcast, or close event.
func (h *Hub) RecordWebSocket(ctx context.Context, event, channel, direction string, size int64, content map[string]any) {
	payload := cloneContent(content)
	payload["event"] = event
	payload["channel"] = channel
	payload["direction"] = direction
	payload["size_bytes"] = size
	h.recordTyped(ctx, TypeWebSocket, []string{"websocket:" + event}, payload)
}

// RecordPerformance records a named performance span.
func (h *Hub) RecordPerformance(ctx context.Context, name string, duration time.Duration, content map[string]any) {
	payload := cloneContent(content)
	payload["name"] = name
	payload["duration_ms"] = milliseconds(duration)
	h.recordTyped(ctx, TypePerformance, []string{"performance:" + name}, payload)
}

// RecordMetric records an application metric sample.
func (h *Hub) RecordMetric(ctx context.Context, name string, value float64, unit string, content map[string]any) {
	payload := cloneContent(content)
	payload["name"] = name
	payload["value"] = value
	payload["unit"] = unit
	h.recordTyped(ctx, TypeMetric, []string{"metric:" + name}, payload)
}

// RecordCustom records a framework or application-defined event.
func (h *Hub) RecordCustom(ctx context.Context, name string, content map[string]any) {
	payload := cloneContent(content)
	payload["name"] = name
	h.recordTyped(ctx, TypeCustom, []string{"custom:" + name}, payload)
}

// RecordTopic records a Redpanda/Kafka produce, consume, or commit operation.
func (h *Hub) RecordTopic(ctx context.Context, topic, action string, duration time.Duration, content map[string]any) {
	payload := cloneContent(content)
	payload["topic"] = topic
	payload["action"] = action
	payload["duration_ms"] = milliseconds(duration)
	h.recordTyped(ctx, TypeTopic, []string{"topic:" + topic, "kafka:" + action}, payload)
}

// RecordEvent records a domain or application event.
func (h *Hub) RecordEvent(ctx context.Context, eventType string, payload map[string]any) {
	content := cloneContent(payload)
	content["event_type"] = eventType
	h.recordTyped(ctx, TypeEvent, []string{"event:" + eventType}, content)
}

// RecordNotification records a notification delivery attempt (e.g. email, SMS, OTP).
func (h *Hub) RecordNotification(ctx context.Context, kind string, content map[string]any) {
	payload := cloneContent(content)
	payload["kind"] = kind
	h.recordTyped(ctx, TypeNotification, []string{"notification:" + kind}, payload)
}

func (h *Hub) recordTyped(ctx context.Context, entryType EntryType, tags []string, content map[string]any) {
	if h == nil {
		return
	}
	meta := RequestMetaFrom(ctx)
	h.Record(ctx, Entry{
		Type:          entryType,
		RequestID:     meta.RequestID,
		CorrelationID: meta.CorrelationID,
		Tags:          tags,
		Content:       RedactMap(content),
		CreatedAt:     time.Now().UTC(),
	})
}

func cloneContent(content map[string]any) map[string]any {
	cloned := make(map[string]any, len(content)+5)
	for key, value := range content {
		cloned[key] = value
	}
	return cloned
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration.Microseconds()) / 1000
}

// HTTPTransport records outgoing HTTP calls made by an http.Client.
type HTTPTransport struct {
	Hub  *Hub
	Next http.RoundTripper
}

// RoundTrip implements http.RoundTripper.
func (t HTTPTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	next := t.Next
	if next == nil {
		next = http.DefaultTransport
	}
	started := time.Now()
	response, err := next.RoundTrip(request)
	duration := time.Since(started)
	content := map[string]any{
		"method":      request.Method,
		"url":         redactedURL(request),
		"host":        request.URL.Host,
		"duration_ms": milliseconds(duration),
	}
	if response != nil {
		content["status"] = response.StatusCode
		content["content_length"] = response.ContentLength
	}
	if err != nil {
		content["error"] = err.Error()
	}
	if t.Hub != nil {
		t.Hub.recordTyped(request.Context(), TypeHTTPClient, []string{"http-client:" + request.Method}, content)
	}
	return response, err
}

func redactedURL(request *http.Request) string {
	if request == nil || request.URL == nil {
		return ""
	}
	copyURL := *request.URL
	copyURL.User = nil
	copyURL.RawQuery = ""
	return copyURL.String()
}

// WrapHTTPClient returns a copy of client that records outgoing requests.
func (h *Hub) WrapHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		client = &http.Client{}
	}
	clone := *client
	clone.Transport = HTTPTransport{Hub: h, Next: client.Transport}
	return &clone
}

// Timed records an operation as a performance span when the returned function is called.
func (h *Hub) Timed(ctx context.Context, name string, content map[string]any) func(error) {
	started := time.Now()
	return func(err error) {
		payload := cloneContent(content)
		if err != nil {
			payload["error"] = fmt.Sprint(err)
		}
		h.RecordPerformance(ctx, name, time.Since(started), payload)
	}
}
