package shared

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func InitLogger(serviceName string) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if os.Getenv("ENV") == "dev" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	} else {
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	}

	log.Logger = log.With().Caller().Str("service", serviceName).Logger()
}
