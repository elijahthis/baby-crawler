package cmd

import (
	"context"

	cmdfactory "github.com/elijahthis/baby-crawler/internal/cmdFactory"
	"github.com/rs/zerolog/log"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

var cfg cmdfactory.Config

func newCmdRootCrawler() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "crawler [flags]",
		Short: "Baby Crawler CLI",
		Long:  `Crawl websites seamlessly from the command line.`,
		Example: heredoc.Doc(`
			$ crawler --seed "https://google.com,https://github.com
			$ crawler --redis-addr "redis:6379
		`),
		Annotations: map[string]string{
			"versionInfo": "1.0",
		},
		RunE: func(c *cobra.Command, args []string) error {
			f := cmdfactory.CrawlerNew(&cfg)

			if len(cfg.SeedURLs) > 0 {
				log.Info().Strs("seeds", cfg.SeedURLs).Msg("Seeding Frontier")
				if err := f.Frontier.Push(context.Background(), cfg.SeedURLs, 0); err != nil {
					log.Error().Err(err).Msg("Frontier Push Error")
				}
			}
			// setup coordinator
			f.Coordinator.Run(context.Background())
			return nil
		},
	}

	addCommonFlags(cmd)
	cmd.Flags().StringSliceVar(&cfg.SeedURLs, "seed", []string{}, "Comma-separated list of start URLs")
	cmd.Flags().IntVar(&cfg.CrawlerWorkerCount, "workers", 10, "Number of crawler workers")
	cmd.Flags().IntVar(&cfg.CrawlerMetricsPort, "metrics-port", 9190, "Port for Metrics server")

	cmd.PersistentFlags().Bool("help", false, "Show help for crawler command")
	return cmd
}

func newCmdRootParser() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "parser [flags]",
		Short: "Baby Crawler Parser CLI",
		Long:  `Crawl websites seamlessly from the command line.`,
		Example: heredoc.Doc(`
			$ parser
		`),
		Annotations: map[string]string{
			"versionInfo": "1.0",
		},
		RunE: func(c *cobra.Command, args []string) error {
			f := cmdfactory.ParserNew(&cfg)

			log.Info().Msg("Starting parser")
			f.Coordinator.Run(context.Background())

			return nil
		},
	}

	addCommonFlags(cmd)
	cmd.Flags().IntVar(&cfg.ParserWorkerCount, "workers", 10, "Number of parser workers")
	cmd.Flags().IntVar(&cfg.ParserMetricsPort, "metrics-port", 9191, "Port for Metrics server")
	cmd.Flags().BoolVar(&cfg.CrawlCrossDomain, "cross-domain", false, "Allow Crawler to crawl links across different domains")

	cmd.PersistentFlags().Bool("help", false, "Show help for parser command")
	return cmd
}

var cmdCrawler = newCmdRootCrawler()
var cmdParser = newCmdRootParser()

func addCommonFlags(cmd *cobra.Command) {
	// Redis
	cmd.Flags().StringVar(&cfg.RedisAddr, "redis-addr", "localhost:6379", "Address of Redis server")
	cmd.Flags().StringVar(&cfg.RedisPassword, "redis-pass", "localhost:6379", "Password of Redis server")
	cmd.Flags().IntVar(&cfg.RedisDB, "redis-db", 0, "Redis DB number")

	// MinIO / S3
	cmd.Flags().StringVar(&cfg.S3Endpoint, "s3-endpoint", "http://localhost:9000", "S3 Endpoint URL")
	cmd.Flags().StringVar(&cfg.S3Bucket, "s3-bucket", "crawled-data", "S3 Bucket name")
	cmd.Flags().StringVar(&cfg.S3Region, "s3-region", "us-east-1", "S3 Region")
	cmd.Flags().StringVar(&cfg.S3User, "s3-user", "admin", "S3 Access Key / User")
	cmd.Flags().StringVar(&cfg.S3Password, "s3-pass", "password", "S3 Secret Key / Password")

}

func ExecuteCrawler() {
	if err := cmdCrawler.Execute(); err != nil {
		log.Fatal().Msg("Error while executing crawler")
	}
}

func ExecuteParser() {
	if err := cmdParser.Execute(); err != nil {
		log.Fatal().Msg("Error while executing parser")
	}
}
