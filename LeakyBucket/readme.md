## Leaky Bucket Algorithm

The leaky bucket algorithm shapes traffic by acting like a bucket with a small hole in the bottom-  
requests drip out at a constant rate, while excess traffic that would overflow is rejected.


### How It Works

- The bucket has a fixed capacity (maximum queue size).
- Incoming requests are queued in the bucket.
- Requests leak out at a constant rate regardless of burstiness.
- When the bucket is full, additional requests are dropped.
- If the bucket is empty, it stops leaking.

This ensures a smooth output rate (no bursts) while still absorbing traffic spikes in short time up to the capacity of bucket.


---
### Diagram
![Leaky Bucket Diagram](./images/LeakyBucket.png)

---
### Pros and Cons

**Pros:**

- It smoothens the burtsy traffic.
- Predictable and constant ouput.
- Protects downstream from overload regardless of upstream burstiness.

**Cons:**

- No support for bursts on the egress side (strictly fixed rate).
- Queued requests add latency so when the queue is full, new requests are lost.


---

## Environment Setup

Required env variables:

| Variable          | Description                                           |
| ----------------- | ----------------------------------------------------- |
| `PORT`            | HTTP server port                                      |
| `BUCKET_CAPACITY` | Maximum number of requests that can wait in the queue |
| `LEAK_RATE`       | Requests per second that leak out of the bucket       |
| `METRICS_USER`    | Username for Prometheus metrics Basic Auth (optional) |
| `METRICS_PASS`    | Password for Prometheus metrics Basic Auth (optional) |

### With Docker Compose

```bash
docker compose up --build
```

### Or run locally

```bash
PORT=8080 BUCKET_CAPACITY=10 LEAK_RATE=2 go run .
```

---

### Prometheus Configuration

Save as `prometheus.yml`:

```yaml
global:
  scrape_interval: 10s
  evaluation_interval: 10s

scrape_configs:
  - job_name: "leaky-bucket"
    static_configs:
      - targets: ["api:yourport"]
    metrics_path: /metrics
    honor_timestamps: true # so that it doesn't scrape from it's current time
```

---

### Grafana Dashboard

Import [grafana.json](./grafana.json) for a ready-made dashboard.

---

## Testing Endpoints

### 1. API Request Endpoint

```bash
curl http://localhost:8080/api/request
```

- **HTTP 200 OK** – request queued and will leak out at the configured rate.
- **HTTP 429 Too Many Requests** – bucket is full, request dropped.

### 2. Load Testing with vegeta

Start hammering the endpoint at a rate that exceeds the LEAK_RATE to see the queue fill up and requests being rejected in real time on the Grafana dashboard.

```Bash
echo "GET http://localhost:8080/" | vegeta attack -rate=50 -duration=30s | vegeta report
```

### 3. Prometheus Metrics Endpoint

```bash
curl http://localhost:8080/metrics
```

Key metrics exposed:

| Metric Name                             | Description                                      |
| --------------------------------------- | ------------------------------------------------ |
| `leaky_bucket_requests_processed_total` | Requests that have leaked out and been processed |
| `leaky_bucket_requests_dropped_total`   | Requests rejected because the queue was full     |
| `leaky_bucket_queue_size`               | Current number of requests waiting in the bucket |
| `leaky_bucket_capacity`                 | Defined bucket capacity                          |
| `leaky_bucket_leak_rate_per_sec`        | Defined leak rate (requests / second)            |

⚠️ Note  
This is just my understanding and attempt at implementing the concept and diagram(which was made by me).  
If something’s off in the implementation — well, that’s part of the learning journey 🚀
