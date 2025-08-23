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
			Name: "leaky_bucket_requests_processed_total",
			Help: "Total requests that leaked out of the bucket",
		},
		[]string{"bucket_name"},
	)

	requestsDroppedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "leaky_bucket_requests_dropped_total",
			Help: "Total requests dropped because the bucket was full",
		},
		[]string{"bucket_name"},
	)

	queueSizeGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "leaky_bucket_queue_size",
			Help: "Current number of requests waiting in the bucket",
		},
		[]string{"bucket_name"},
	)

	capacityGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "leaky_bucket_capacity",
			Help: "Maximum number of requests the bucket can hold",
		},
		[]string{"bucket_name"},
	)

	leakRateGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "leaky_bucket_leak_rate_per_sec",
			Help: "Configured leak rate (requests per second)",
		},
		[]string{"bucket_name"},
	)
)

// Wraper for metrics
type MetricsLeakyBucket struct {
	*LeakyBucket
	name string
}

func NewMetricsLeakyBucket(name string, capacity int64, rate float64, unit TimeUnit) (*MetricsLeakyBucket, error) {
	lb, err := NewLeakyBucket(capacity, rate, unit)
	if err != nil {
		return nil, err
	}
	mlb := &MetricsLeakyBucket{LeakyBucket: lb, name: name}

	capacityGauge.WithLabelValues(name).Set(float64(capacity))
	leakRateGauge.WithLabelValues(name).Set(mlb.LeakRate())

	return mlb, nil
}

func (mlb *MetricsLeakyBucket) Allow(n int) bool {
	ok := mlb.LeakyBucket.Allow(n)

	if ok {
		requestsProcessedTotal.WithLabelValues(mlb.name).Add(float64(n))
	} else {
		requestsDroppedTotal.WithLabelValues(mlb.name).Add(float64(n))
	}

	_, dropped, queueSize := mlb.Stats()
	_ = dropped                                             // silence unused warning
	requestsProcessedTotal.WithLabelValues(mlb.name).Add(0) // keeps label set alive
	queueSizeGauge.WithLabelValues(mlb.name).Set(queueSize)

	return ok
}

func main() {
	// Create a leaky bucket: 10 req capacity, 2 req/sec leak
	godotenv.Load()

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
	bucketCapacity, err := strconv.ParseInt(bucketCapacityStr, 10, 64)
	if err != nil {
		fmt.Printf("Invalid BUCKET_CAPACITY: %v\n", err)
		os.Exit(1)
	}

	leakRateStr := os.Getenv("LEAK_RATE")
	if leakRateStr == "" {
		fmt.Println("LEAK_RATE env variable is required")
		os.Exit(1)
	}
	leakRate, err := strconv.ParseFloat(leakRateStr, 64)
	if err != nil {
		fmt.Printf("Invalid LEAK_RATE: %v\n", err)
		os.Exit(1)
	}
	http.Handle("/metrics", promhttp.Handler())

	apiBucket, err := NewMetricsLeakyBucket("api_rate_limit", bucketCapacity, leakRate, PerSecond)
	if err != nil {
		log.Fatal(err)
	}

	// single endpoint only
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

	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	log.Printf("Server starting on :%s ...\n", port)
	log.Printf("Metrics: http://localhost:%s/metrics\n", port)
	log.Printf("Test this endpoint: http://localhost:%s/api/request\n", port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
