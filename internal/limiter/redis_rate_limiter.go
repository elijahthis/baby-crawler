package limiter

import (
	"context"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRateLimiter struct {
	client *redis.Client
	delay  time.Duration
}

func NewRedisRateLimiter(rdb *redis.Client, delay time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: rdb,
		delay:  delay,
	}
}

func (rl *RedisRateLimiter) Wait(ctx context.Context, domain string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			success, err := rl.client.SetNX(ctx, domain, 1, rl.delay).Result()
			if err != nil {
				return err
			}
			if success {
				return nil
			}

			ttl, err := rl.client.TTL(ctx, domain).Result()
			if err != nil {
				return err
			}

			if ttl <= 0 {
				time.Sleep(1 * time.Second)
			} else {
				jitter := time.Duration(rand.Int63n(int64(ttl / 10)))
				time.Sleep(ttl + jitter)
			}
		}
	}
}
