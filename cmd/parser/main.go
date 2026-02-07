package main

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/elijahthis/baby-crawler/internal/frontier"
	"github.com/elijahthis/baby-crawler/internal/parser"
	"github.com/elijahthis/baby-crawler/internal/shared"
	"github.com/elijahthis/baby-crawler/internal/storage"
	"github.com/redis/go-redis/v9"
)

func main() {
	shared.InitLogger("parser")
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "password", // no password set
		DB:       0,
	})

	fr := frontier.NewRedisFrontier(rdb)

	htmlParser := parser.NewHTMLParser()

	store, err := storage.NewS3Storage(context.Background(), "crawled-data", "http://localhost:9000", "admin", "password")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize storage")
	}

	// setup coordinator
	coord := parser.NewService(fr, store, htmlParser, 10)
	log.Info().Msg("Starting parser")
	coord.Run(context.Background())
}
