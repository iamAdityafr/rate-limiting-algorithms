package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics
var (
	tokensProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "token_bucket_tokens_processed_total",
			Help: "Number of tokens processed",
		},
		[]string{"bucket_name"},
	)

	tokensRejectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "token_bucket_tokens_rejected_total",
			Help: "Number of tokens rejected",
		},
		[]string{"bucket_name"},
	)

	availableTokensGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "token_bucket_available_tokens",
			Help: "Number of tokens available",
		},
		[]string{"bucket_name"},
	)

	capacityGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "token_bucket_capacity",
			Help: "Maximum capacity",
		},
		[]string{"bucket_name"},
	)

	fillRateGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "token_bucket_fill_rate",
			Help: "Fill rate ", // tokens per second
		},
		[]string{"bucket_name"},
	)

	usagePercentGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "token_bucket_usage_percent",
			Help: "Percentage of bucket used",
		},
		[]string{"bucket_name"},
	)
)

// Wraping token bucket with metrics
type MetricsTokenBucket struct {
	*TokenBucket
	name string
}

func NewMetricsTokenBucket(name string, capacity int, tokens, fillRate float64) *MetricsTokenBucket {
	tb, err := NewTokenBucket(capacity, tokens, fillRate)
	if err != nil {
		log.Fatalf("Failed to create bucket: %v", err)
	}

	mtb := &MetricsTokenBucket{
		TokenBucket: tb,
		name:        name,
	}

	// Starts metrics
	mtb.updateMetrics()
	capacityGauge.WithLabelValues(name).Set(float64(capacity))
	fillRateGauge.WithLabelValues(name).Set(fillRate)

	return mtb
}

func (mtb *MetricsTokenBucket) Allow(n int) bool {
	allowed := mtb.TokenBucket.Allow(n)

	if allowed {
		tokensProcessedTotal.WithLabelValues(mtb.name).Add(float64(n))
	} else {
		tokensRejectedTotal.WithLabelValues(mtb.name).Inc()
	}

	mtb.updateMetrics()
	return allowed
}

func (mtb *MetricsTokenBucket) updateMetrics() {
	availableTokensGauge.WithLabelValues(mtb.name).Set(mtb.AvailableTokens())
	usagePercent := 100 * (1 - mtb.AvailableTokens()/float64(mtb.capacity))
	usagePercentGauge.WithLabelValues(mtb.name).Set(usagePercent)
}

func main() {
	// Creating single bucket
	port := os.Getenv("PORT")
	if port == "" {
		fmt.Println("PORT env variable is required")
		os.Exit(1)
	}

	bucketCapacityStr := os.Getenv("BUCKET_CAPACITY")
	if bucketCapacityStr == "" {
		fmt.Println("BUCKET_CAPACITY env variable is required")
		os.Exit(1)
	}
	bucketCapacity, err := strconv.Atoi(bucketCapacityStr)
	if err != nil {
		fmt.Println("Invalid BUCKET_CAPACITY:", err)
		os.Exit(1)
	}

	fillRateStr := os.Getenv("FILL_RATE")
	if fillRateStr == "" {
		fmt.Println("FILL_RATE env variable is required")
		os.Exit(1)
	}
	fillRate, err := strconv.ParseFloat(fillRateStr, 64)
	if err != nil {
		fmt.Println("Invalid FILL_RATE:", err)
		os.Exit(1)
	}

	apiBucket := NewMetricsTokenBucket("api_rate_limit", bucketCapacity, float64(bucketCapacity), fillRate)

	// Adding single endpoint
	http.HandleFunc("/api/request", func(w http.ResponseWriter, r *http.Request) {
		allowed := apiBucket.Allow(1)
		status := http.StatusTooManyRequests
		msg := "Rate limit exceeded"
		if allowed {
			status = http.StatusOK
			msg = "Request allowed"
		}
		log.Printf("Method: %s, Path: %s, Status: %d", r.Method, r.URL.Path, status)
		w.WriteHeader(status)
		fmt.Fprintln(w, msg)
	})

	// userEnv := os.Getenv("METRICS_USER")
	// passEnv := os.Getenv("METRICS_PASS")
	http.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("Server starting on :%s ...\n", port)

	log.Printf("Test this endpoint: http://localhost:%s/api/request\n", port)

	// Doing graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
