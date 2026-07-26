package microscope

import (
	"context"
	"net"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// RedisHook records go-redis commands without capturing command arguments.
type RedisHook struct {
	hub *Hub
}

// NewRedisHook creates a go-redis instrumentation hook.
func NewRedisHook(hub *Hub) *RedisHook {
	return &RedisHook{hub: hub}
}

// DialHook instruments Redis connection attempts.
func (h *RedisHook) DialHook(next goredis.DialHook) goredis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		started := time.Now()
		connection, err := next(ctx, network, addr)
		h.hub.RecordRedis(ctx, "DIAL", time.Since(started), err, map[string]any{
			"network": network,
			"address": addr,
		})
		return connection, err
	}
}

// ProcessHook instruments a single Redis command.
func (h *RedisHook) ProcessHook(next goredis.ProcessHook) goredis.ProcessHook {
	return func(ctx context.Context, command goredis.Cmder) error {
		started := time.Now()
		err := next(ctx, command)
		h.hub.RecordRedis(ctx, strings.ToUpper(command.Name()), time.Since(started), err, map[string]any{
			"argument_count": len(command.Args()) - 1,
		})
		return err
	}
}

// ProcessPipelineHook instruments a Redis pipeline as a single signal.
func (h *RedisHook) ProcessPipelineHook(next goredis.ProcessPipelineHook) goredis.ProcessPipelineHook {
	return func(ctx context.Context, commands []goredis.Cmder) error {
		started := time.Now()
		err := next(ctx, commands)
		names := make([]string, 0, len(commands))
		for _, command := range commands {
			names = append(names, strings.ToUpper(command.Name()))
		}
		h.hub.RecordRedis(ctx, "PIPELINE", time.Since(started), err, map[string]any{
			"commands": names,
			"count":    len(commands),
		})
		return err
	}
}
