package pkg

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Close() error
}

type Redis struct {
	Client *redis.Client
	Tracer trace.Tracer
}

func NewRedis(cfg RedisConfig) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &Redis{Client: client, Tracer: otel.Tracer("zpay-backend.redis")}, nil
}

func (r *Redis) Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd {
	_, redisSpan := r.Tracer.Start(ctx, "redis.set")
	defer redisSpan.End()
	return r.Client.Set(ctx, key, value, expiration)
}

func (r *Redis) Get(ctx context.Context, key string) *redis.StringCmd {
	_, redisSpan := r.Tracer.Start(ctx, "redis.get")
	defer redisSpan.End()
	return r.Client.Get(ctx, key)
}

func (r *Redis) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return r.Client.Exists(ctx, keys...)
}

func (r *Redis) Close() error {
	return r.Client.Close()
}
