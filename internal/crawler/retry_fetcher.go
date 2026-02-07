package crawler

import (
	"context"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/elijahthis/baby-crawler/internal/shared"
	"github.com/rs/zerolog/log"
)

const maxRetries = 10
const baseTime = 1 * time.Second
const maxBackoff = 32 * time.Second

type RetryFetcher struct {
	Base    shared.Fetcher
	Retries int
}

func (rf *RetryFetcher) Fetch(ctx context.Context, url string) (shared.FetchResult, error) {
	var lastErr error

	for i := 0; i < rf.Retries; i++ {
		log.Info().Msgf("Attempt %d: ", i+1)
		resp, err := rf.Base.Fetch(ctx, url)
		if err == nil {
			if resp.StatusCode == http.StatusOK {
				return resp, nil
			}
			// resp.Body.Close()
		}
		log.Error().Err(err).Msg("Retry Error")

		lastErr = err

		nextBackoff := generateExponentialBackoff(i)
		select {
		case <-ctx.Done():
			return shared.FetchResult{}, ctx.Err()
		case <-time.After(nextBackoff):
		}
	}

	return shared.FetchResult{}, lastErr
}

func generateExponentialBackoff(i int) time.Duration {
	backoff := time.Duration(math.Min(float64(baseTime)*math.Pow(2, float64(i)), float64(maxBackoff)))
	jitter := time.Duration(rand.Float64() * float64(backoff) * 0.5)
	nextBackoff := backoff + jitter

	return nextBackoff
}
