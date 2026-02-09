package crawler

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/elijahthis/baby-crawler/internal/frontier"
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
}

func NewCoordinator(f frontier.Frontier, fetch shared.Fetcher, l shared.RateLimiter, s shared.Storage, r *robots.RobotsChecker, workerCount int) *Coordinator {
	return &Coordinator{
		frontier: f,
		fetcher:  fetch,
		limiter:  l,
		storage:  s,
		robots:   r,
		workers:  workerCount,
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

			if err := c.limiter.Wait(ctx, domain); err != nil {
				itemLog.Error().Err(err).Msg("Rate Limiter error: ")
				continue
			}

			if !c.robots.IsAllowed(urlTarget.URL) {
				itemLog.Error().Msgf("Blocked by robots.txt: %s", urlTarget.URL)
				c.frontier.Complete(ctx, urlTarget.ID)
				continue
			}

			func() {
				itemLog.Printf("Worker %d fetching: %s", id, urlTarget.URL)
				resp, err := c.fetcher.Fetch(ctx, urlTarget.URL)
				if err != nil {
					itemLog.Error().Err(err).Msgf("Worker %d Failed Final", id)
					// retry logic. Dead letter queue
					if dlqErr := c.frontier.PushDLQ(ctx, urlTarget, err.Error()); dlqErr != nil {
						itemLog.Error().Err(dlqErr).Msg("Failed to push to DLQ:")
						return
					}
					return
				}
				if resp.Body == nil {
					itemLog.Error().Msgf("Worker %d error: Body is nil for %s", id, urlTarget.URL)
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
				if err := c.storage.Save(ctx, s3Key, bodyBytes); err != nil {
					itemLog.Error().Err(err).Msgf("Worker %d storage error", id)
					c.frontier.PushDLQ(ctx, urlTarget, "Storage Upload Failed")
					// maybe stop? will come back to this
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

				// HandleParsed(parsed, urlTarget.URL)
			}()

			c.frontier.Complete(ctx, urlTarget.ID)
		}
	}
}

func HandleParsed(parsedData shared.ParsedData, link string) error {
	urlObj, err := url.Parse(link)
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing link: %s\n", link)
		return err
	}
	filePath := urlObj.Path
	fileName := strings.ReplaceAll(filePath, "/", "_")

	folderPath := "/Users/elijahoyerinde/Documents/baby-crawler/data"
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		log.Error().Err(err).Msgf("Error creating folder: %s\n", folderPath)
		return err
	}
	fullPath := filepath.Join(folderPath, fileName)

	file, err := os.Create(fullPath)
	if err != nil {
		log.Error().Err(err).Msgf("Error creating file: %s\n", fullPath)
	}
	defer file.Close()

	if _, err := file.WriteString(parsedData.Text); err != nil {
		log.Error().Err(err).Msgf("Error writing to file: %s\n", fullPath)
		return err
	}

	return nil
}
