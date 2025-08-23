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
			Name: "sliding_window_requests_processed_total",
			Help: "Total requests processed",
		},
		[]string{"window_name"},
	)

	requestsRejectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sliding_window_requests_rejected_total",
			Help: "Total requests rejected",
		},
		[]string{"window_name"},
	)

	slidingCountGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sliding_window_current_sliding_count",
			Help: "Current sliding window request count",
		},
		[]string{"window_name"},
	)

	maxRequestsGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sliding_window_max_requests",
			Help: "Maximum requests allowed in window",
		},
		[]string{"window_name"},
	)
)

// Wrapping with prometheus metrics
type MetricsSlidingWindow struct {
	*SlidingWindow
	name string
}

func NewMetricsSlidingWindow(name string, windowSize time.Duration, maxRequests int64) (*MetricsSlidingWindow, error) {
	sw, err := NewSlidingWindow(windowSize, maxRequests)
	if err != nil {
		return nil, err
	}
	msw := &MetricsSlidingWindow{SlidingWindow: sw, name: name}
	maxRequestsGauge.WithLabelValues(name).Set(float64(maxRequests))
	return msw, nil
}

func (msw *MetricsSlidingWindow) Allow(n int) bool {
	ok := msw.SlidingWindow.Allow(n)
	if ok {
		requestsProcessedTotal.WithLabelValues(msw.name).Add(float64(n))
	} else {
		requestsRejectedTotal.WithLabelValues(msw.name).Add(float64(n))
	}
	allowed, denied, sliding := msw.DetailedStats()
	_ = allowed
	_ = denied
	slidingCountGauge.WithLabelValues(msw.name).Set(sliding)
	return ok
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
		fmt.Println("WINDOW_SIZE is requried")
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

	apiRateLimiter, err := NewMetricsSlidingWindow("api_rate_limit", windowSize, maxRequests)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/api/request", func(w http.ResponseWriter, r *http.Request) {
		allowed := apiRateLimiter.Allow(1)
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
