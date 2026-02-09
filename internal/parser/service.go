package parser

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/elijahthis/baby-crawler/internal/frontier"
	"github.com/elijahthis/baby-crawler/internal/shared"
	"github.com/rs/zerolog/log"
)

type Service struct {
	frontier frontier.Frontier
	storage  shared.Storage
	parser   shared.Parser
	workers  int
}

func NewService(f frontier.Frontier, s shared.Storage, p shared.Parser, w int) *Service {
	return &Service{
		frontier: f,
		storage:  s,
		parser:   p,
		workers:  w,
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

			bodyBytes, err := s.storage.Load(ctx, msg.S3Key)
			if err != nil {
				msgLog.Error().Err(err).Msgf("Parser %d: Failed to load S3 key %s", id, msg.S3Key)
				continue
			}

			// process result
			parsed, err := s.parser.Parse(ctx, bytes.NewReader(bodyBytes))
			if err != nil {
				msgLog.Error().Err(err).Msgf("Parser %d parse error", id)
				continue
			}

			if len(parsed.Links) > 0 {
				var absoluteLinks []string
				for _, link := range parsed.Links {
					abs, err := shared.ResolveURL(msg.URL, link)
					if err != nil {
						continue
					}

					isSameDomain, err := shared.CompareDomains(msg.URL, abs)
					if err != nil {
						continue
					}

					if isSameDomain {
						absoluteLinks = append(absoluteLinks, abs)
					}
				}
				if len(absoluteLinks) > 0 {
					if err := s.frontier.Push(ctx, absoluteLinks, msg.Depth+1); err != nil {
						msgLog.Error().Err(err).Msg("Frontier Push Error")
					}
				}
			}
			msgLog.Info().Msgf("Parser %d: Processed %s (%d links)", id, msg.URL, len(parsed.Links))
		}
	}
}
