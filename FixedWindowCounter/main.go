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

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus metrics
var (
	requestsProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fixed_window_requests_processed_total",
			Help: "Number of requests processed",
		},
		[]string{"window_name"},
	)

	requestsRejectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fixed_window_requests_rejected_total",
			Help: "Number of requests rejected",
		},
		[]string{"window_name"},
	)

	currentRequestsGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fixed_window_current_requests",
			Help: "Current number of requests in window",
		},
		[]string{"window_name"},
	)

	maxRequestsGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fixed_window_max_requests",
			Help: "Maximum requests allowed in window",
		},
		[]string{"window_name"},
	)

	timeUntilResetGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "fixed_window_time_until_reset_seconds",
			Help: "Time until current window resets",
		},
		[]string{"window_name"},
	)

	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "fixed_window_request_duration_seconds",
			Help:    "Time taken to process requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"window_name", "status"},
	)
)

// Wrapping with Prometheus metrics
type MetricsFixedWindowCounter struct {
	*FixedWindowCounter
	name string
}

func NewMetricsFixedWindowCounter(name string, windowSize time.Duration, maxRequests int64) (*MetricsFixedWindowCounter, error) {
	fwc, err := NewFixedWindowCounter(windowSize, maxRequests)
	if err != nil {
		return nil, err
	}

	mfwc := &MetricsFixedWindowCounter{
		FixedWindowCounter: fwc,
		name:               name,
	}
	maxRequestsGauge.WithLabelValues(name).Set(float64(mfwc.MaxRequests))
	timeUntilResetGauge.WithLabelValues(name).Set(mfwc.TimeUntilReset().Seconds())

	return mfwc, nil
}

func (mfwc *MetricsFixedWindowCounter) Allow(n int) bool {
	start := time.Now()
	allowed := mfwc.FixedWindowCounter.Allow(n)
	duration := time.Since(start).Seconds()

	status := "rejected"
	if allowed {
		status = "allowed"
		requestsProcessedTotal.WithLabelValues(mfwc.name).Add(float64(n))
	} else {
		requestsRejectedTotal.WithLabelValues(mfwc.name).Add(float64(n))
	}

	requestDuration.WithLabelValues(mfwc.name, status).Observe(duration)

	currentCount, _, _ := mfwc.FixedWindowCounter.Stats()
	currentRequestsGauge.WithLabelValues(mfwc.name).Set(float64(currentCount))
	timeUntilResetGauge.WithLabelValues(mfwc.name).Set(mfwc.TimeUntilReset().Seconds())

	return allowed
}

func main() {
	godotenv.Load()
	port := os.Getenv("PORT")
	if port == "" {
		fmt.Println("PORT is required")
		os.Exit(1)
	}
	windowSizestr := os.Getenv("WINDOW_SIZE")
	if windowSizestr == "" {
		fmt.Println("WINDOW_SIZE is required")
		os.Exit(1)
	}
	windowSize, err := time.ParseDuration(windowSizestr)
	if err != nil {
		fmt.Printf("Invalid WINDOW_SIZE: %v", err)
		os.Exit(1)
	}
	maxRequestsStr := os.Getenv("MAX_REQUESTS")
	if maxRequestsStr == "" {
		log.Fatal("MAX_REQUESTS is required")
	}
	maxRequests, err := strconv.ParseInt(maxRequestsStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid MAX_REQUESTS: %v", err)
	}

	// Metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	apiRateLimit, err := NewMetricsFixedWindowCounter("api_rate_limit", windowSize, maxRequests)
	if err != nil {
		log.Fatal("Couldn't create Fixed Window Counter:", err)
	}

	// testing endpoint
	http.HandleFunc("/api/request", func(w http.ResponseWriter, r *http.Request) {
		allowed := apiRateLimit.Allow(1)
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

	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting server on: %s ....\n", port)
	log.Printf("Metrics available at http://localhost:%s/metrics", port)
	log.Printf("http://localhost:%s/api/request ", port)

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
