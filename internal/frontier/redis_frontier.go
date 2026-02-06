package frontier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/elijahthis/baby-crawler/internal/shared"
	"github.com/redis/go-redis/v9"
)

type RedisFrontier struct {
	client     *redis.Client
	queueKey   string
	visitedKey string
}

func NewRedisFrontier(rdb *redis.Client) *RedisFrontier {
	return &RedisFrontier{
		client:     rdb,
		queueKey:   "crawler:queue",
		visitedKey: "crawler:visited",
	}
}

func (f *RedisFrontier) Push(ctx context.Context, urls []string, depth int) error {
	// pipe := f.client.Pipeline()
	for _, u := range urls {
		// fmt.Printf("Pushing %s to fr\n", u)
		isNew, err := f.client.SAdd(ctx, f.visitedKey, u).Result()
		if err != nil {
			fmt.Printf("There was an error: %v\n", err)
			return err
		}

		// count, err := f.client.SCard(ctx, f.visitedKey).Result()
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// fmt.Printf("set has %d members\n", count)
		// f.client.Del(ctx, f.visitedKey)
		if isNew == 1 {
			target := shared.URLTarget{
				ID:    u,
				URL:   u,
				Depth: depth,
			}

			targetJson, err := json.Marshal(target)
			if err != nil {
				return err
			}

			if err := f.client.RPush(ctx, f.queueKey, targetJson).Err(); err != nil {
				return err
			}
			fmt.Printf("Pushed %s to fr\n", u)
		}
	}
	return nil
}

func (f *RedisFrontier) Pop(ctx context.Context) (shared.URLTarget, error) {
	var target shared.URLTarget
	targetJson, err := f.client.LPop(ctx, f.queueKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return target, ErrQueueEmpty
		}
		return target, err
	}

	if err := json.Unmarshal([]byte(targetJson), &target); err != nil {
		return target, err
	}

	return target, nil
}

func (f *RedisFrontier) Complete(ctx context.Context, id string) error {
	return nil
}
