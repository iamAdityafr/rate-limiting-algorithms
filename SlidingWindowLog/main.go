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
			Name: "sliding_window_log_requests_processed_total",
			Help: "Number of requests processed",
		},
		[]string{"window_name"},
	)

	requestsRejectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sliding_window_log_requests_rejected_total",
			Help: "Number of requests rejected",
		},
		[]string{"window_name"},
	)

	currentCountGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sliding_window_log_current_count",
			Help: "Current number of requests in the sliding window",
		},
		[]string{"window_name"},
	)

	maxRequestsGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sliding_window_log_max_requests",
			Help: "Maximum requests in window",
		},
		[]string{"window_name"},
	)

	windowSizeGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sliding_window_log_window_size_seconds",
			Help: "Window size",
		},
		[]string{"window_name"},
	)
)

type MetricsSlidingWindowLog struct {
	*SlidingWindowLog
	name string
}

func NewMetricsSlidingWindowLog(name string, windowSize time.Duration, maxRequests int64) (*MetricsSlidingWindowLog, error) {
	sw, err := NewSlidingWindowLog(windowSize, maxRequests)
	if err != nil {
		return nil, err
	}
	msw := &MetricsSlidingWindowLog{SlidingWindowLog: sw, name: name}
	maxRequestsGauge.WithLabelValues(name).Set(float64(maxRequests))
	windowSizeGauge.WithLabelValues(name).Set(windowSize.Seconds())
	return msw, nil
}

func (msw *MetricsSlidingWindowLog) Allow(n int) bool {
	ok := msw.SlidingWindowLog.Allow(n)
	if ok {
		requestsProcessedTotal.WithLabelValues(msw.name).Add(float64(n))
	} else {
		requestsRejectedTotal.WithLabelValues(msw.name).Add(float64(n))
	}
	current, _, _ := msw.Stats()
	currentCountGauge.WithLabelValues(msw.name).Set(float64(current))
	return ok
}

func main() {
	godotenv.Load()
	port := os.Getenv("PORT")
	if port == "" {
		fmt.Println("PORT not found")
		os.Exit(1)
	}
	windowSizestr := os.Getenv("WINDOW_SIZE")
	if windowSizestr == "" {
		fmt.Println("WINDOW_SIZE not found")
		os.Exit(1)
	}
	windowSize, err := time.ParseDuration(windowSizestr)
	if err != nil {
		fmt.Printf("WINDOW_SIZE is invalid: %v", err)
		os.Exit(1)

	}
	maxRequeststr := os.Getenv("MAX_REQUESTS")
	if maxRequeststr == "" {
		fmt.Println("MAX_REQUESTS not found")
		os.Exit(1)
	}
	maxRequest, err := strconv.ParseInt(maxRequeststr, 10, 64)
	if err != nil {
		fmt.Printf("MAX_REQUESTS is invalid: %v", err)
		os.Exit(1)

	}

	apiRateLimiter, err := NewMetricsSlidingWindowLog("api", windowSize, maxRequest)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/api/request", func(w http.ResponseWriter, r *http.Request) {
		allowed := apiRateLimiter.Allow(1)
		status := http.StatusTooManyRequests
		msg := "rate limit exceeded"
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
		IdleTimeout:  50 * time.Second,
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
