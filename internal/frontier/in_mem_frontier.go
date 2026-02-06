package frontier

import (
	"context"
	"errors"
	"sync"

	"github.com/elijahthis/baby-crawler/internal/shared"
)

var ErrQueueEmpty = errors.New("Frontier queue is empty")

type InMemFrontier struct {
	mu      sync.Mutex
	pending []shared.URLTarget
	visited map[string]struct{}
}

func NewInMemFrontier() *InMemFrontier {
	return &InMemFrontier{
		pending: make([]shared.URLTarget, 0),
		visited: make(map[string]struct{}),
	}
}

func (f *InMemFrontier) Push(ctx context.Context, urls []string, depth int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, u := range urls {
		if _, exists := f.visited[u]; exists {
			continue
		}

		f.pending = append(f.pending, shared.URLTarget{
			ID:    u,
			URL:   u,
			Depth: depth,
		})
		f.visited[u] = struct{}{}
	}
	return nil

}

func (f *InMemFrontier) Pop(ctx context.Context) (shared.URLTarget, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.pending) == 0 {
		return shared.URLTarget{}, ErrQueueEmpty
	}

	target := f.pending[0]
	f.pending = f.pending[1:]

	return target, nil
}

func (f *InMemFrontier) Complete(ctx context.Context, id string) error {
	return nil
}
