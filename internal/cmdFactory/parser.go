package cmdfactory

import (
	"strconv"

	"github.com/elijahthis/baby-crawler/internal/frontier"
	"github.com/elijahthis/baby-crawler/internal/metrics"
	"github.com/elijahthis/baby-crawler/internal/parser"
	"github.com/elijahthis/baby-crawler/internal/shared"
	"github.com/redis/go-redis/v9"
)

type parserFactory struct {
	*commonFactory
	Frontier frontier.Frontier
	Parser   shared.Parser

	Store            shared.Storage
	Coordinator      *parser.Service
	CrawlCrossDomain bool
}

func ParserNew(cfg *Config) *parserFactory {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword, // no password set
		DB:       cfg.RedisDB,
	})

	met := metrics.NewMetrics()
	go metrics.StartNewMetricsServer(":" + strconv.Itoa(cfg.ParserMetricsPort))

	f := &parserFactory{
		commonFactory: &commonFactory{
			RDB:         rdb,
			userAgent:   "BabyCrawler/1.0",
			workerCount: cfg.ParserWorkerCount,
		},
		CrawlCrossDomain: cfg.CrawlCrossDomain,
	}

	f.Frontier = newFrontier(f.commonFactory)
	f.Parser = newParser()
	f.Metrics = met

	f.Store = newStorage(cfg)
	f.Coordinator = newService(f)

	return f
}

func newParser() shared.Parser {
	htmlParser := parser.NewHTMLParser()

	return htmlParser
}

func newService(f *parserFactory) *parser.Service {
	coord := parser.NewService(f.Frontier, f.Store, f.Parser, f.workerCount, f.Metrics, f.CrawlCrossDomain)
	return coord
}
