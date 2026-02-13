package shared

import (
	"context"
	"io"
	"time"
)

// interfaces
type Fetcher interface {
	Fetch(ctx context.Context, url string) (FetchResult, error)
}
type Parser interface {
	Parse(ctx context.Context, r io.Reader) (ParsedData, error)
}
type RateLimiter interface {
	Wait(ctx context.Context, domain string, delay time.Duration) error
}
type Storage interface {
	Save(ctx context.Context, key string, data []byte) error
	Load(ctx context.Context, key string) ([]byte, error)
}

// structs
type URLTarget struct {
	ID    string
	URL   string
	Depth int
}

type FetchResult struct {
	StatusCode  int
	Body        io.ReadCloser
	ContentType string
}

type ParsedData struct {
	Text  string
	Links []string
}

type CrawlResult struct {
	URL     string `json:"url"`
	S3Key   string `json:"s3_key"`
	Depth   int    `json:"depth"`
	SavedAt string `json:"saved_at"`
}
