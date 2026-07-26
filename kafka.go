package microscope

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaWriter instruments a kafka-go writer with Redpanda topic signals.
type KafkaWriter struct {
	Writer *kafka.Writer
	Hub    *Hub
}

// NewKafkaWriter wraps writer with topic instrumentation.
func NewKafkaWriter(writer *kafka.Writer, hub *Hub) *KafkaWriter {
	return &KafkaWriter{Writer: writer, Hub: hub}
}

// WriteMessages writes messages and records one signal for the produced batch.
func (w *KafkaWriter) WriteMessages(ctx context.Context, messages ...kafka.Message) error {
	started := time.Now()
	err := w.Writer.WriteMessages(ctx, messages...)
	topic := w.Writer.Topic
	if topic == "" && len(messages) > 0 {
		topic = messages[0].Topic
	}
	var bytes int
	for _, message := range messages {
		bytes += len(message.Key) + len(message.Value)
	}
	w.Hub.RecordTopic(ctx, topic, "produce", time.Since(started), map[string]any{
		"message_count": len(messages),
		"size_bytes":    bytes,
		"error":         errorString(err),
	})
	return err
}

// Close closes the wrapped writer.
func (w *KafkaWriter) Close() error {
	return w.Writer.Close()
}

// KafkaReader instruments kafka-go fetch and commit operations.
type KafkaReader struct {
	Reader *kafka.Reader
	Hub    *Hub
}

// NewKafkaReader wraps reader with topic instrumentation.
func NewKafkaReader(reader *kafka.Reader, hub *Hub) *KafkaReader {
	return &KafkaReader{Reader: reader, Hub: hub}
}

// FetchMessage fetches a message and records its topic, partition, and offset.
func (r *KafkaReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	started := time.Now()
	message, err := r.Reader.FetchMessage(ctx)
	topic := message.Topic
	if topic == "" {
		topic = r.Reader.Config().Topic
	}
	r.Hub.RecordTopic(ctx, topic, "consume", time.Since(started), map[string]any{
		"partition":  message.Partition,
		"offset":     message.Offset,
		"size_bytes": len(message.Key) + len(message.Value),
		"error":      errorString(err),
	})
	return message, err
}

// CommitMessages commits messages and records the acknowledgement operation.
func (r *KafkaReader) CommitMessages(ctx context.Context, messages ...kafka.Message) error {
	started := time.Now()
	err := r.Reader.CommitMessages(ctx, messages...)
	topic := r.Reader.Config().Topic
	if topic == "" && len(messages) > 0 {
		topic = messages[0].Topic
	}
	r.Hub.RecordTopic(ctx, topic, "commit", time.Since(started), map[string]any{
		"message_count": len(messages),
		"error":         errorString(err),
	})
	return err
}

// Close closes the wrapped reader.
func (r *KafkaReader) Close() error {
	return r.Reader.Close()
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
