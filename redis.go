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
		content := map[string]any{
			"argument_count": len(command.Args()) - 1,
		}
		if h.hub != nil && !h.hub.RedactSensitive() && len(command.Args()) > 1 {
			content["args"] = command.Args()[1:]
		}
		h.hub.RecordRedis(ctx, strings.ToUpper(command.Name()), time.Since(started), err, content)
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
		content := map[string]any{
			"commands": names,
			"count":    len(commands),
		}
		if h.hub != nil && !h.hub.RedactSensitive() {
			args := make([][]any, 0, len(commands))
			for _, command := range commands {
				if len(command.Args()) > 1 {
					args = append(args, command.Args()[1:])
				} else {
					args = append(args, nil)
				}
			}
			content["args"] = args
		}
		h.hub.RecordRedis(ctx, "PIPELINE", time.Since(started), err, content)
		return err
	}
}
