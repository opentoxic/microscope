package microscope

import (
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

// Integration wires microscope into a host Go application with minimal setup.
type Integration struct {
	cfg    Config
	active bool
	tracer *QueryTracer
	hub    *Hub
}

// NewIntegration prepares microscope for the given app environment and config.
// When active, call QueryTracer before creating the pgx pool, then Bind after the pool exists.
func NewIntegration(appEnv string, cfg Config) *Integration {
	active := Enabled(appEnv, cfg)
	var tracer *QueryTracer
	if active {
		tracer = NewQueryTracer()
	}
	return &Integration{
		cfg:    cfg,
		active: active,
		tracer: tracer,
	}
}

// Active reports whether microscope is enabled for this process.
func (i *Integration) Active() bool {
	return i.active
}

// Config returns the integration configuration.
func (i *Integration) Config() Config {
	return i.cfg
}

// QueryTracer returns a pgx query tracer to attach before pool creation.
func (i *Integration) QueryTracer() pgx.QueryTracer {
	if i.tracer == nil {
		return nil
	}
	return i.tracer
}

// Bind creates the Hub using an existing pool. Safe to call when inactive (returns nil).
func (i *Integration) Bind(pool *pgxpool.Pool, log *slog.Logger) *Hub {
	if !i.active || pool == nil {
		return nil
	}
	i.hub = NewWithTracer(pool, i.cfg, log, i.tracer)
	return i.hub
}

// Hub returns the bound hub, if any.
func (i *Integration) Hub() *Hub {
	return i.hub
}

// TeeSlog returns a logger that tees records to microscope.
func (i *Integration) TeeSlog(log *slog.Logger) *slog.Logger {
	if i.hub == nil || log == nil {
		return log
	}
	return slog.New(NewSlogHandler(log.Handler(), i.hub))
}

// RedisHook returns a go-redis hook for the bound hub, or nil when inactive.
func (i *Integration) RedisHook() goredis.Hook {
	if i.hub == nil {
		return nil
	}
	return NewRedisHook(i.hub)
}

// WrapEventPublisher records domain events through the bound hub.
func (i *Integration) WrapEventPublisher(inner EventPublisher) EventPublisher {
	if i.hub == nil || inner == nil {
		return inner
	}
	return WrapEventPublisher(i.hub, inner)
}

// WrapOTPNotifier records OTP notifications through the bound hub.
func (i *Integration) WrapOTPNotifier(inner OTPNotifier) OTPNotifier {
	if i.hub == nil || inner == nil {
		return inner
	}
	return WrapOTPNotifier(i.hub, inner)
}

// ClusterHealthChecker probes Kafka/Redpanda brokers and records cluster signals.
func (i *Integration) ClusterHealthChecker(brokers []string) *ClusterHealthChecker {
	return NewClusterHealthChecker(brokers, i.hub)
}

// Close shuts down background hub workers.
func (i *Integration) Close() {
	if i.hub != nil {
		i.hub.Close()
	}
}
