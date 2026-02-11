package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type PrometheusMetrics struct {
	// Crawler
	PagesFetched  *prometheus.CounterVec
	FetchDuration *prometheus.HistogramVec
	RobotsBlocked *prometheus.CounterVec
	FetchErrors   *prometheus.CounterVec
	QueueDepth    *prometheus.GaugeVec

	// Parser
	PagesParsed      *prometheus.CounterVec
	LinksFound       *prometheus.CounterVec
	ParseDuration    *prometheus.HistogramVec
	S3AccessDuration *prometheus.HistogramVec
	S3AccessErrors   *prometheus.CounterVec
}

func NewMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		PagesFetched: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "crawler_pages_fetched_total",
				Help: "Total number of pages fetched successfully",
			},
			[]string{"status_code"},
		),
		FetchDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "crawler_fetch_duration_seconds",
				Help: "Time taken to download a page",
			},
			[]string{},
		),
		RobotsBlocked: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "crawler_robots_blocked_total",
				Help: "Number of requests blocked by robots.txt",
			},
			[]string{},
		),
		FetchErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "crawler_fetch_errors_total",
				Help: "Total number of fetch errors found",
			},
			[]string{"status_code", "type"},
		),
		QueueDepth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "crawler_queue_depth_total",
				Help: "Current number of items in the Redis queue",
			}, []string{"queue_name"},
		),

		PagesParsed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "parser_pages_processed_total",
				Help: "Total HTML pages parsed",
			},
			[]string{},
		),
		LinksFound: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "parser_links_extracted_total",
				Help: "Total new links found on pages",
			}, []string{},
		),
		ParseDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "parser_processing_duration_seconds",
				Help: "Time taken to parse HTML",
			}, []string{},
		),
		S3AccessDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "s3_interaction_duration_seconds",
				Help: "Time taken to read/write to S3",
			}, []string{"operation"},
		), // label: "upload" or "download"
		S3AccessErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "s3_interaction_errors_total",
				Help: "Total number of S3 read/write errors",
			}, []string{"operation"},
		), // label: "upload" or "download"

	}
}

func StartNewMetricsServer(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	log.Info().Msgf("Metrics server starting on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Error().Err(err).Msg("Metrics server failed")
	}
}

func (m *PrometheusMetrics) MonitorQueueDepth(ctx context.Context, rdb *redis.Client, queues map[string]string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for label, key := range queues {
				val, err := rdb.LLen(ctx, key).Result()
				if err != nil {
					log.Error().Err(err).Msgf("Failed to monitor queue: %s", key)
					continue
				}

				m.QueueDepth.WithLabelValues(label).Set(float64(val))
			}
		}
	}
}
