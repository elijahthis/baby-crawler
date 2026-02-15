package crawler

import (
	"context"
	"errors"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/elijahthis/baby-crawler/internal/frontier"
	"github.com/elijahthis/baby-crawler/internal/metrics"
	"github.com/elijahthis/baby-crawler/internal/robots"
	"github.com/elijahthis/baby-crawler/internal/shared"
)

type Coordinator struct {
	frontier frontier.Frontier
	fetcher  shared.Fetcher
	parser   shared.Parser
	limiter  shared.RateLimiter
	storage  shared.Storage
	robots   *robots.RobotsChecker
	workers  int
	metrics  *metrics.PrometheusMetrics
}

func NewCoordinator(f frontier.Frontier, fetch shared.Fetcher, l shared.RateLimiter, s shared.Storage, r *robots.RobotsChecker, workerCount int, met *metrics.PrometheusMetrics) *Coordinator {
	return &Coordinator{
		frontier: f,
		fetcher:  fetch,
		limiter:  l,
		storage:  s,
		robots:   r,
		workers:  workerCount,
		metrics:  met,
	}
}

func (c *Coordinator) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			c.worker(ctx, workerID)
		}(i)
	}

	wg.Wait()
	log.Info().Msg("All workers shut down cleanly")
}

func (c *Coordinator) worker(ctx context.Context, id int) {
	logger := log.With().Int("worker_id", id).Logger()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			urlTarget, err := c.frontier.Pop(ctx)
			if err != nil {
				if errors.Is(err, frontier.ErrQueueEmpty) {
					time.Sleep(500 * time.Millisecond) // Backoff so we don't spam the CPU
					// log.Printf("Error: %s", err.Error())
					continue
				}
				logger.Error().Err(err).Msgf("Worker %d frontier error:", id)
				continue
			}

			itemLog := logger.With().Str("url", urlTarget.URL).Logger()

			domain, err := shared.GetDomain(urlTarget.URL)
			if err != nil {
				itemLog.Error().Err(err).Msgf("Invalid URL in queue: %s", urlTarget.URL)
				c.frontier.Complete(ctx, urlTarget.ID)
				continue
			}

			if !c.robots.IsAllowed(urlTarget.URL) {
				itemLog.Error().Msgf("Blocked by robots.txt: %s", urlTarget.URL)
				c.metrics.RobotsBlocked.WithLabelValues().Inc()
				c.frontier.Complete(ctx, urlTarget.ID)
				continue
			}

			delay := c.robots.GetCrawlDelay(urlTarget.URL)

			defaultDelay := 1 * time.Second
			if delay < defaultDelay {
				delay = defaultDelay
			}

			if err := c.limiter.Wait(ctx, domain, delay); err != nil {
				itemLog.Error().Err(err).Msg("Rate Limiter error: ")
				continue
			}

			func() {
				itemLog.Printf("Worker %d fetching: %s", id, urlTarget.URL)

				start := time.Now()
				resp, err := c.fetcher.Fetch(ctx, urlTarget.URL)
				duration := time.Since(start).Seconds()
				c.metrics.FetchDuration.WithLabelValues().Observe(duration)

				if err != nil {
					itemLog.Error().Err(err).Msgf("Worker %d Failed Final", id)

					c.metrics.FetchErrors.WithLabelValues(strconv.Itoa(resp.StatusCode), "network").Inc()
					// retry logic. Dead letter queue
					if dlqErr := c.frontier.PushDLQ(ctx, urlTarget, err.Error()); dlqErr != nil {
						itemLog.Error().Err(dlqErr).Msg("Failed to push to DLQ:")
						return
					}
					return
				}

				c.metrics.PagesFetched.WithLabelValues(strconv.Itoa(resp.StatusCode)).Inc()

				if resp.Body == nil {
					itemLog.Error().Msgf("Worker %d error: Body is nil for %s", id, urlTarget.URL)

					c.metrics.FetchErrors.WithLabelValues(strconv.Itoa(resp.StatusCode), "nil body").Inc()

					c.frontier.PushDLQ(ctx, urlTarget, "Nil Body Response")
					return
				}
				defer resp.Body.Close()

				// save to s3
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					itemLog.Error().Err(err).Msgf("Worker %d read error", id)
					return
				}

				s3Key := shared.CleanKey(urlTarget.URL)

				startS3 := time.Now()
				errS3 := c.storage.Save(ctx, s3Key, bodyBytes)
				durationS3 := time.Since(startS3).Seconds()

				c.metrics.S3AccessDuration.WithLabelValues("upload").Observe(durationS3)

				if errS3 != nil {
					itemLog.Error().Err(err).Msgf("Worker %d storage error: %s", id, errS3)

					c.metrics.S3AccessErrors.WithLabelValues("upload").Inc()

					c.frontier.PushDLQ(ctx, urlTarget, "Storage Upload Failed")
					// maybe stop? will come back to this
					return
				}

				// Push to Parser Queue
				msg := shared.CrawlResult{
					URL:   urlTarget.URL,
					S3Key: s3Key,
					Depth: urlTarget.Depth,
				}
				if err := c.frontier.PushToParser(ctx, msg); err != nil {
					itemLog.Error().Err(err).Msgf("Failed to push to parser queue")
				} else {
					itemLog.Info().Msgf("Worker %d: Fetched & Pushed %s", id, urlTarget.URL)
				}

			}()

			c.frontier.Complete(ctx, urlTarget.ID)
		}
	}
}
