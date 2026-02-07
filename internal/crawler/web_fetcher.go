package crawler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/elijahthis/baby-crawler/internal/shared"
	"github.com/rs/zerolog/log"
)

type WebFetcher struct {
	client    *http.Client
	userAgent string
}

func NewWebFetcher(userAgent string, timeout time.Duration) *WebFetcher {
	return &WebFetcher{
		client: &http.Client{
			Timeout: timeout,
		},
		userAgent: userAgent,
	}
}

func (f *WebFetcher) Fetch(ctx context.Context, url string) (shared.FetchResult, error) {
	log.Info().Msgf("Fetching %s", url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return shared.FetchResult{}, err
	}

	req.Header.Set("User-Agent", f.userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return shared.FetchResult{}, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		err := fmt.Errorf("non-200 status code: %d", resp.StatusCode)
		log.Error().Err(err)
		return shared.FetchResult{}, err
	}

	return shared.FetchResult{
		StatusCode:  resp.StatusCode,
		Body:        resp.Body,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}
