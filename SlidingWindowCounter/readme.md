# Sliding-Window Counter Algorithm

The **sliding-window counter algorithm** is a hybrid approach that combines the fixed-window counter and the sliding-window log.

### How It Works

1.  Two consecutive windows are kept in memory:

    - the current (partially elapsed) window
    - the previous (fully elapsed) window

2.  The algorithm smoothly weights the count from the previous window according to how much of the current window has already elapsed.

    ```
    slidingCount = currentCount + (1 – elapsedRatio) * previousCount
    ```

3.  A request is allowed only if `slidingCount + n ≤ maxRequests`.

4.  **Important to Note:** This algorithm uses a calculated **estimation** of the request count based on the portion of the previous window. Not strict precise but it effectively prevents the boundary spike problem seen in the fixed-window method.
---

### Diagram

![Sliding Window Counter Diagram](./images/SlidingWindowCounter.png)


---

### Key Characteristics

- Handle short bursts of traffic up to **maxRequests** limit.
- Output rate is much **smoother** than the fixed-window approach.


---

### Pros & Cons

#### Pros

- Effectively solves that "boundary spike" problem that arises int the fixed-window algorithm.
- Predictable average rate of requests over the specified window.

#### Cons

- It estimates the request rate by combining counts from the current and previous windows with weighting which is approximation and not exact.
- Since it’s an approximation, users might occasionally see inconsistent throttling (in other words, getting limited even though they feel under quota)
-  Needs to calculate weighted sums across windows. For very high request volumes this adds CPU overhead compared to fixed window.
---

## Environment Setup

Required environment variables:

| Variable       | Description                                            |
| -------------- | ------------------------------------------------------ |
| `PORT`         | HTTP server port                                       |
| `WINDOW_SIZE`  | Duration of the sliding window (e.g. **60s**)      |
| `MAX_REQUESTS` | Maximum requests allowed **within any rolling window** |

### With Docker Compose

```bash
docker compose up --build
```

### Or run locally

```bash
PORT=8080 WINDOW_SIZE=1m MAX_REQUESTS=20 go run .
```

---

### Prometheus Configuration

Save as `prometheus.yml`:

```yaml
global:
  scrape_interval: 10s
  evaluation_interval: 10s

scrape_configs:
  - job_name: "sliding-window-counter"
    static_configs:
      - targets: ["api:yourport"]
    metrics_path: /metrics
    honor_timestamps: true
```

---

### Grafana Dashboard

Import [grafana.json](https://www.google.com/search?q=./grafana.json) for a ready-made dashboard.

### Demo Output

Here is an example of what a load test looks like on the dashboard, showing the real-time weighted count and processed requests.

---

## Testing Endpoints

### 1\. API Request Endpoint

```bash
curl http://localhost:8080/api/request
```

- **HTTP 200 OK** – request allowed inside the rolling limit.
- **HTTP 429 Too Many Requests** – rolling limit exceeded.

### 2\. Load Testing with `vegeta`

Start hammering the endpoint and watch the real-time metrics and rolling count on the Grafana dashboard.

```bash
echo "GET http://localhost:8080/" | vegeta attack -rate=50 -duration=30s | vegeta report
```

### 3\. Prometheus Metrics Endpoint

```bash
curl http://localhost:8080/metrics
```

Key metrics exposed:

| Metric Name                               | Meaning                                            |
| ----------------------------------------- | -------------------------------------------------- |
| `sliding_window_requests_processed_total` | Requests accepted so far                           |
| `sliding_window_requests_rejected_total`  | Requests rejected by the rolling limit             |
| `sliding_window_current_sliding_count`    | Real-time weighted count inside the sliding window |
| `sliding_window_max_requests`             | Configured limit per window                        |


⚠️ Note  
This is just my understanding and attempt at implementing the concept and diagram(which was made by me).  
If something’s off in the implementation — well, that’s part of the learning journey 🚀
