package parser

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/elijahthis/baby-crawler/internal/frontier"
	"github.com/elijahthis/baby-crawler/internal/metrics"
	"github.com/elijahthis/baby-crawler/internal/shared"
	"github.com/rs/zerolog/log"
)

type Service struct {
	frontier         frontier.Frontier
	storage          shared.Storage
	parser           shared.Parser
	workers          int
	metrics          *metrics.PrometheusMetrics
	crawlCrossDomain bool
}

func NewService(f frontier.Frontier, s shared.Storage, p shared.Parser, w int, met *metrics.PrometheusMetrics, crawlCrossDomain bool) *Service {
	return &Service{
		frontier:         f,
		storage:          s,
		parser:           p,
		workers:          w,
		metrics:          met,
		crawlCrossDomain: crawlCrossDomain,
	}
}

func (s *Service) Run(ctx context.Context) {
	var wg sync.WaitGroup

	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			s.worker(ctx, workerID)
		}(i)
	}

	wg.Wait()
	log.Info().Msg("All Parser workers shut down cleanly")
}

func (s *Service) worker(ctx context.Context, id int) {
	logger := log.With().Int("parser_id", id).Logger()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := s.frontier.PopFromParser(ctx)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			msgLog := logger.With().Str("url", msg.URL).Str("s3_key", msg.S3Key).Logger()

			startS3 := time.Now()
			bodyBytes, err := s.storage.Load(ctx, msg.S3Key)
			durationS3 := time.Since(startS3).Seconds()

			s.metrics.S3AccessDuration.WithLabelValues("download").Observe(durationS3)

			if err != nil {
				msgLog.Error().Err(err).Msgf("Parser %d: Failed to load S3 key %s", id, msg.S3Key)
				s.metrics.S3AccessErrors.WithLabelValues("download").Inc()
				continue
			}

			// process result
			start := time.Now()
			parsed, err := s.parser.Parse(ctx, bytes.NewReader(bodyBytes))
			if err != nil {
				msgLog.Error().Err(err).Msgf("Parser %d parse error", id)
				continue
			}
			duration := time.Since(start).Seconds()
			s.metrics.ParseDuration.WithLabelValues().Observe(duration)

			if len(parsed.Links) > 0 {
				var absoluteLinks []string
				for _, link := range parsed.Links {
					abs, err := shared.ResolveURL(msg.URL, link)
					if err != nil {
						continue
					}

					if !s.crawlCrossDomain {
						isSameDomain, err := shared.CompareDomains(msg.URL, abs)
						if err != nil || !isSameDomain {
							continue
						}
					}
					absoluteLinks = append(absoluteLinks, abs)
				}
				if len(absoluteLinks) > 0 {
					s.metrics.LinksFound.WithLabelValues().Add(float64(len(absoluteLinks)))

					if err := s.frontier.Push(ctx, absoluteLinks, msg.Depth+1); err != nil {
						msgLog.Error().Err(err).Msg("Frontier Push Error")
					}
				}
			}
			s.metrics.PagesParsed.WithLabelValues().Inc()

			msgLog.Info().Msgf("Parser %d: Processed %s (%d links)", id, msg.URL, len(parsed.Links))
		}
	}
}
