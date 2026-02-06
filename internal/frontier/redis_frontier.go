package frontier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/elijahthis/baby-crawler/internal/shared"
	"github.com/redis/go-redis/v9"
)

type RedisFrontier struct {
	client     *redis.Client
	queueKey   string
	visitedKey string
	dlqKey     string
}

type DeadLetter struct {
	Target shared.URLTarget `json:"target"`
	Error  string           `json:"error"`
	Time   string           `json:"time"`
}

func NewRedisFrontier(rdb *redis.Client) *RedisFrontier {
	return &RedisFrontier{
		client:     rdb,
		queueKey:   "crawler:queue",
		visitedKey: "crawler:visited",
		dlqKey:     "crawler:dlq",
	}
}

func (f *RedisFrontier) Push(ctx context.Context, urls []string, depth int) error {
	// pipe := f.client.Pipeline()
	for _, u := range urls {
		isNew, err := f.client.SAdd(ctx, f.visitedKey, u).Result()
		if err != nil {
			fmt.Printf("There was an error: %v\n", err)
			return err
		}

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

func (f *RedisFrontier) PushDLQ(ctx context.Context, item shared.URLTarget, errReason string) error {
	dl := DeadLetter{
		Target: item,
		Error:  errReason,
		Time:   time.Now().Format(time.RFC3339),
	}

	dlData, err := json.Marshal(dl)
	if err != nil {
		return err
	}

	return f.client.RPush(ctx, f.dlqKey, dlData).Err()
}
