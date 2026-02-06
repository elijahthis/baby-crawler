package crawler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/elijahthis/baby-crawler/internal/frontier"
	"github.com/elijahthis/baby-crawler/internal/shared"
)

type Coordinator struct {
	frontier frontier.Frontier
	fetcher  shared.Fetcher
	parser   shared.Parser
	limiter  shared.RateLimiter
	storage  shared.Storage
	workers  int
}

func NewCoordinator(f frontier.Frontier, fetch shared.Fetcher, p shared.Parser, l shared.RateLimiter, s shared.Storage, workerCount int) *Coordinator {
	return &Coordinator{
		frontier: f,
		fetcher:  fetch,
		parser:   p,
		limiter:  l,
		storage:  s,
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
	log.Printf("All workers shut down cleanly")
}

func (c *Coordinator) worker(ctx context.Context, id int) {
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
				log.Printf("Worker %d frontier error: %v", id, err)
				continue
			}

			domain, err := getDomain(urlTarget.URL)
			if err != nil {
				log.Printf("Invalid URL in queue: %s", urlTarget.URL)
				c.frontier.Complete(ctx, urlTarget.ID)
				continue
			}

			if err := c.limiter.Wait(ctx, domain); err != nil {
				log.Printf("Rate Limiter error: %v", err)
				continue
			}

			func() {
				log.Printf("Worker %d fetching: %s", id, urlTarget.URL)
				resp, err := c.fetcher.Fetch(ctx, urlTarget.URL)
				if err != nil {
					log.Printf("Worker %d error: %v", id, err)
					// retry logic. Dead letter queue
					return
				}
				if resp.Body == nil {
					log.Printf("Worker %d error: Body is nil for %s", id, urlTarget.URL)
					return
				}
				defer resp.Body.Close()

				// save to s3
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Printf("Worker %d read error: %v", id, err)
					return
				}

				if err := c.storage.Save(ctx, urlTarget.URL, bodyBytes); err != nil {
					log.Printf("Worker %d storage error: %v", id, err)
					// maybe stop? will come back to this
				}

				// process result
				parsed, err := c.parser.Parse(ctx, bytes.NewReader(bodyBytes))
				if err != nil {
					log.Printf("Worker %d parse error: %v", id, err)
					return
				}

				if len(parsed.Links) > 0 {
					var absoluteLinks []string
					for _, link := range parsed.Links {
						abs, err := ResolveURL(urlTarget.URL, link)
						if err != nil {
							continue
						}

						isSameDomain, err := compareDomains(urlTarget.URL, abs)
						if err != nil {
							continue
						}

						if isSameDomain {
							absoluteLinks = append(absoluteLinks, abs)
						}
					}
					if len(absoluteLinks) > 0 {
						c.frontier.Push(ctx, absoluteLinks, urlTarget.Depth+1)
					}
				}

				// HandleParsed(parsed, urlTarget.URL)
			}()

			c.frontier.Complete(ctx, urlTarget.ID)
		}
	}
}

func ResolveURL(parent, link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", nil
	}

	base, err := url.Parse(parent)
	if err != nil {
		return "", nil
	}

	return base.ResolveReference(u).String(), nil
}

func getDomain(link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", err
	}
	return u.Host, nil
}

func compareDomains(parent, child string) (bool, error) {
	parentDomain, err := getDomain(parent)
	if err != nil {
		return false, err
	}
	childDomain, err := getDomain(child)
	if err != nil {
		return false, err
	}

	return parentDomain == childDomain, nil
}

func HandleParsed(parsedData shared.ParsedData, link string) error {
	urlObj, err := url.Parse(link)
	if err != nil {
		return err
	}
	filePath := urlObj.Path
	fileName := strings.ReplaceAll(filePath, "/", "_")

	folderPath := "/Users/elijahoyerinde/Documents/baby-crawler/data"
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		return err
	}
	fullPath := filepath.Join(folderPath, fileName)

	file, err := os.Create(fullPath)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
	}
	defer file.Close()

	if _, err := file.WriteString(parsedData.Text); err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
	}

	return nil
}
