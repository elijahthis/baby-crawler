package frontier

import (
	"context"

	"github.com/elijahthis/baby-crawler/internal/shared"
)

type Frontier interface {
	Push(ctx context.Context, urls []string, depth int) error
	Pop(ctx context.Context) (shared.URLTarget, error)
	Complete(ctx context.Context, id string) error
}
