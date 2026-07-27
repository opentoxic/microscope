package microscope

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

const clusterDialTimeout = 3 * time.Second

// ClusterHealthChecker probes Kafka/Redpanda broker connectivity.
type ClusterHealthChecker struct {
	brokers []string
	hub     *Hub
}

// NewClusterHealthChecker returns a checker when brokers are configured, or nil otherwise.
func NewClusterHealthChecker(brokers []string, hub *Hub) *ClusterHealthChecker {
	if len(brokers) == 0 {
		return nil
	}
	return &ClusterHealthChecker{brokers: brokers, hub: hub}
}

// Check dials the first reachable broker.
func (c *ClusterHealthChecker) Check(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("cluster health checker not configured")
	}
	started := time.Now()
	if len(c.brokers) == 0 {
		return fmt.Errorf("no brokers configured")
	}

	var lastErr error
	for _, broker := range c.brokers {
		dialCtx, cancel := context.WithTimeout(ctx, clusterDialTimeout)
		conn, err := kafka.DialContext(dialCtx, "tcp", broker)
		cancel()
		if err != nil {
			lastErr = err
			continue
		}
		_ = conn.SetDeadline(time.Now().Add(clusterDialTimeout))
		partitions, metadataErr := conn.ReadPartitions()
		topics := uniqueKafkaTopics(partitions)
		_ = conn.Close()
		if c.hub != nil {
			content := map[string]any{
				"broker":      broker,
				"brokers":     len(c.brokers),
				"topics":      topics,
				"topic_count": len(topics),
			}
			if metadataErr != nil {
				content["metadata_error"] = metadataErr.Error()
			}
			c.hub.RecordTopic(ctx, "__cluster__", "discover", time.Since(started), content)
		}
		return nil
	}
	if lastErr != nil {
		if c.hub != nil {
			c.hub.RecordTopic(ctx, "__cluster__", "connect", time.Since(started), map[string]any{
				"brokers": len(c.brokers),
				"error":   lastErr.Error(),
			})
		}
		return fmt.Errorf("all brokers unreachable: %w", lastErr)
	}
	return fmt.Errorf("no brokers configured")
}

func uniqueKafkaTopics(partitions []kafka.Partition) []string {
	seen := make(map[string]struct{})
	topics := make([]string, 0)
	for _, partition := range partitions {
		if partition.Topic == "" {
			continue
		}
		if _, exists := seen[partition.Topic]; exists {
			continue
		}
		seen[partition.Topic] = struct{}{}
		topics = append(topics, partition.Topic)
	}
	return topics
}
