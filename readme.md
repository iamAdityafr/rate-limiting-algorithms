# Rate Limiting Algorithms

> Go • Prometheus • Grafana 
 
This repo contains **five independent reference implementations** of the most common rate-limiting algorithms.

## Algorithms

- [Fixed Window Counter](https://github.com/iamAdityafr/rate-limiting-algorithms/tree/main/FixedWindowCounter)
- [Sliding Window Counter](https://github.com/iamAdityafr/rate-limiting-algorithms/tree/main/SlidingWindowCounter)
- [Sliding Window Log](https://github.com/iamAdityafr/rate-limiting-algorithms/tree/main/SlidingWindowLog)
- [Token Bucket](https://github.com/iamAdityafr/rate-limiting-algorithms/tree/main/TokenBucket)
- [Leaky Bucket](https://github.com/iamAdityafr/rate-limiting-algorithms/tree/main/LeakyBucket)

**Click on an algorithm for details**

## 🚀 Quick start


- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/) — for containerized deployments
- [vegeta](https://github.com/tsenart/vegeta) — for load testing
- [Prometheus](https://prometheus.io/download/) — metrics collection
- [Grafana](https://grafana.com/grafana/download) — metrics visualization

For detailed instructions and configuration for each algorithm, please refer to the `README.md` file in its respective directory.

1. Navigate to the directory of the algorithm you wish to run (e.g., `cd FixedWindowCounter`).
2. Start the services: `docker-compose up -d`
3. Open **Grafana** at http://localhost:3000 and import the dashboard from `grafana.json`.
4. Start hammering the endpoint and watch the metrics in real time:

```bash
# Example load test with vegeta
echo "GET http://localhost:8080/" | vegeta attack -rate=50 -duration=30s | vegeta report
```

## Folder Structure

```
RateLimiter/
.
├── FixedWindowCounter
│   ├── docker-compose.yml
│   ├── Dockerfile
│   ├── fixedwindow.go
│   ├── fixedwindow_test.go
│   ├── go.mod
│   ├── go.sum
│   ├── grafana.json
│   ├── main.go
│   ├── prometheus.yml
│   └── readme.md
├── images
│   ├── FixedWindow.png
│   ├── LeakyBucket.png
│   ├── SlidingWindowCounter.png
│   ├── SlidingWindowLog.png
│   └── tokenBucket.png
├── LeakyBucket
│   ├── docker-compose.yml
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   ├── grafana.json
│   ├── leakybucket.go
│   ├── leakybucket_test.go
│   ├── main.go
│   ├── prometheus.yml
│   └── readme.md
├── productionreadydockerfile.txt
├── readme.md
├── SlidingWindowCounter
│   ├── docker-compose.yml
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   ├── grafana.json
│   ├── main.go
│   ├── prometheus.yml
│   ├── readme.md
│   ├── slidingWindowCounter.go
│   └── slidingWindowCounter_test.go
├── SlidingWindowLog
│   ├── Deque.go
│   ├── docker-compose.yml
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   ├── grafana.json
│   ├── main.go
│   ├── prometheus.yml
│   ├── readme.md
│   ├── slidingWindowLog.go
│   └── slidingWindowLog_test.go
└── tokenBucket
    ├── docker-compose.yml
    ├── Dockerfile
    ├── go.mod
    ├── go.sum
    ├── grafana.json
    ├── main.go
    ├── prometheus.yml
    ├── readme.md
    ├── tokenBucket.go
    └── tokenBucket_test.go
```

## For more instructions on each algorithm, refer to the `README.md` file in its respective folder.

⚠️ Note  
This is just my understanding and attempt at implementing the concept and diagram(which was made by me).  
If something’s off in the implementation — well, that’s part of the learning journey 🚀

---
