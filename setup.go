package microscope

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

// SetupOptions configures one-call microscope integration.
type SetupOptions struct {
	AppEnv string
	Config Config
	DSN    string
	Logger *slog.Logger

	Redis   goredis.UniversalClient
	Brokers []string

	// AutoMigrate runs embedded migrations when active. Nil defaults to cfg.AutoMigrate.
	AutoMigrate *bool

	// PoolConfig adjusts pgx pool settings before creation.
	PoolConfig func(*pgxpool.Config)
}

// SetupResult holds wired microscope resources.
type SetupResult struct {
	*Integration
	Pool   *pgxpool.Pool
	Logger *slog.Logger
	Kafka  *ClusterHealthChecker
}

// Close shuts down the hub and database pool.
func (r *SetupResult) Close() {
	if r == nil {
		return
	}
	if r.Integration != nil {
		r.Integration.Close()
	}
	if r.Pool != nil {
		r.Pool.Close()
	}
}

// Setup wires microscope in-process: pool, hub, logger, and optional instrumentation.
func Setup(ctx context.Context, opts SetupOptions) (*SetupResult, error) {
	if opts.DSN == "" {
		return nil, errors.New("microscope: DSN is required")
	}

	cfg := MergeConfig(ConfigFromEnv(), opts.Config)

	integ := NewIntegration(opts.AppEnv, cfg)
	poolCfg, err := pgxpool.ParseConfig(opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("microscope: parse dsn: %w", err)
	}
	integ.ConfigurePool(poolCfg)
	if opts.PoolConfig != nil {
		opts.PoolConfig(poolCfg)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("microscope: create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("microscope: ping database: %w", err)
	}

	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}

	result := &SetupResult{
		Integration: integ,
		Pool:        pool,
		Logger:      log,
	}

	if !integ.Active() {
		return result, nil
	}

	autoMigrate := cfg.AutoMigrate
	if opts.AutoMigrate != nil {
		autoMigrate = *opts.AutoMigrate
	}
	if autoMigrate {
		if err := MigrateUp(ctx, pool); err != nil {
			result.Close()
			return nil, fmt.Errorf("microscope: migrate: %w", err)
		}
	}

	integ.Bind(pool, log)
	result.Logger = integ.TeeSlog(log)

	if opts.Redis != nil {
		if hook := integ.RedisHook(); hook != nil {
			opts.Redis.AddHook(hook)
		}
	}
	if len(opts.Brokers) > 0 {
		result.Kafka = integ.ClusterHealthChecker(opts.Brokers)
	}

	return result, nil
}

// ConfigurePool attaches the query tracer before pool creation.
func (i *Integration) ConfigurePool(cfg *pgxpool.Config) {
	if i == nil || cfg == nil {
		return
	}
	if tracer := i.QueryTracer(); tracer != nil {
		cfg.ConnConfig.Tracer = tracer
	}
}
