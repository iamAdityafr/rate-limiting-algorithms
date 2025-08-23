# Token Bucket Algorithm

The **token bucket algorithm** is a widely used rate limiting algorithm and easy to implement.

### How It Works

- Think of this as like a bucket with some maximum capacity.
- Tokens are added at a fixed rate over time.
- A request is allowed only when enough tokens are available in the bucket.
- If there are not enough tokens, the request is rejected. (Optionally we can make it wait until tokens are available - queue behavior).

**Important to Note :** The Leaky Bucket smoothens the egress rate but in Token bucket allows for high rate consumption for a short period obviously as long as tokens are available. It maintains the overall average rate over time.

---

### Diagram

![Token Bucket Diagram](./images/tokenBucket.png)

---

### Pros and Cons

#### Pros

-  The algorithm allows a burst of traffic for short period and the bucket can store tokens for later use.
-  The core logic is simple and easy to implement.

#### Cons

-  Requires additional coordination (using a central data store like Redis) to maintain a single and consistent state across a distributed fleet.
- Two parameters in the algorithm are bucket size and token refill rate. It might be challenging to tune them properly. This means that if the bucket is too small, you deny legitimate bursts, if the bucket is too big you allow bigger bursts than desired, if the refill rate is too low then the throughput suffers and if it’s too high then the limiter becomes meaningless.

---

## Environment Setup

Required env variables:

| Variable          | Description                                           |
| ----------------- | ----------------------------------------------------- |
| `PORT`            | HTTP server port                              |
| `BUCKET_CAPACITY` | Maximum number of tokens in the bucket                |
| `FILL_RATE`       | Tokens per second to refill the bucket                |
| `METRICS_USER`    | Username for Prometheus metrics Auth (optional) |
| `METRICS_PASS`    | Password for Prometheus metrics Auth (optional) |

### With Docker Compose:

```bash
docker compose up --build
```

### Or locally:

```bash
PORT=yourport BUCKET_CAPACITY=10 FILL_RATE=1 go run .
```

---

### Prometheus Configuration

Save as `prometheus.yml`:

```yaml
global:
  scrape_interval: 10s
  evaluation_interval: 10s

scrape_configs:
  - job_name: "token-bucket"
    static_configs:
      - targets: ["api:yourport"]
    metrics_path: /metrics
    honor_timestamps: true # so that it doesn't scrape from current time
```

---

### Grafana Dashboard

Import [grafana.json](https://www.google.com/search?q=./grafana.json) for a ready-made dashboard.

### Demo Output

Here is an example of a load test on the dashboard, showing the bucket's fill and usage percentage over time.

---

### Testing Endpoints

#### 1\. API Request Endpoint

```bash
curl http://localhost:8080/api/request
```

- **HTTP 200 OK** – Request allowed if tokens are available.
- **HTTP 429 Too Many Requests** – Rate limit exceeded if no tokens are left.

#### 2\. Load Testing with `vegeta`

Start hammering the endpoint at a rate that exceeds the **FILL_RATE** to see the available tokens decrease and requests being rejected in real time on the Grafana dashboard.

```bash
echo "GET http://localhost:8080/" | vegeta attack -rate=50 -duration=30s | vegeta report
```

#### 3\. Prometheus Metrics Endpoint

```bash
curl -u $METRICS_USER:$METRICS_PASS http://localhost:8080/metrics
```

Key metrics exposed:

| Metric Name                           | Description                                                       |
| ------------------------------------- | ----------------------------------------------------------------- |
| `token_bucket_tokens_processed_total` | Total number of requests successfully processed (tokens consumed) |
| `token_bucket_tokens_rejected_total`  | Total number of requests rejected due to insufficient tokens      |
| `token_bucket_available_tokens`       | Current number of tokens available in the bucket                  |
| `token_bucket_fill_rate`              | Rate at which tokens are added to the bucket (tokens per second)  |
| `token_bucket_usage_percent`          | Percentage of bucket usage relative to its capacity               |


⚠️ Note  
This is just my understanding and attempt at implementing the concept and diagram(which was made by me).  
If something’s off in the implementation — well, that’s part of the learning journey 🚀
