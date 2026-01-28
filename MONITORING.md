# Monitoring Stack Setup Guide

This guide explains how to set up and use the complete monitoring stack for the microservices demo using Docker Desktop.

## Overview

The monitoring stack provides three layers of visibility:

1. **Container Metrics** - CPU, memory, network from cAdvisor
2. **Application Metrics** - Request rates, latency, errors from instrumented services
3. **Infrastructure Metrics** - Host system metrics from node-exporter

## Architecture

```
┌─────────────────┐
│  Microservices  │
│   (12 services) │
└────────┬────────┘
         │ /metrics endpoints
         ▼
    ┌─────────┐       ┌──────────┐
    │ cAdvisor│◄──────┤  Docker  │
    └────┬────┘       └──────────┘
         │
         │ scrape
         ▼
   ┌──────────────┐      ┌─────────┐
   │  Prometheus  │◄─────┤ Grafana │
   │ (Time-series)│      │(Dashboards)│
   └──────────────┘      └─────────┘
```

## Prerequisites

- Docker Desktop running on Windows
- Microservices-demo already running via `docker-compose up`
- At least 4GB RAM allocated to Docker

## Quick Start

### 1. Start the Monitoring Stack

```powershell
# From the project root
docker-compose -f docker-compose.monitoring.yml up -d
```

This starts:

- **Prometheus** on `http://localhost:9090`
- **Grafana** on `http://localhost:3001`
- **cAdvisor** on `http://localhost:8080`
- **node-exporter** on `http://localhost:9100`

### 2. Rebuild Instrumented Services

Two services have been instrumented with Prometheus metrics:

**Frontend (Go):**

```powershell
cd src/frontend
go mod tidy
docker-compose up -d --build frontend
```

**Checkout Service (Go):**

```powershell
cd src/checkoutservice
go mod tidy
docker-compose up -d --build checkoutservice
```

### 3. Access Grafana

1. Open `http://localhost:3001`
2. Login with:
   - **Username:** `admin`
   - **Password:** `admin`
3. Navigate to **Dashboards** → **Browse**
4. Select:
   - **Container Metrics Overview** - CPU/memory per container
   - **Application Metrics - Frontend & Checkout** - Request rates and latency

## What You Can See Now

### Container-Level (No Code Changes Required)

**Metrics Available:**

- `container_cpu_usage_seconds_total` - CPU usage per container
- `container_memory_usage_bytes` - Memory consumption
- `container_network_receive_bytes_total` - Network RX
- `container_network_transmit_bytes_total` - Network TX

**Dashboard:** Container Metrics Overview

### Application-Level (Frontend & Checkout)

**Frontend Metrics (`/metrics` on port 8080):**

- `http_requests_total{method, handler, status_code}` - Request count
- `http_request_duration_seconds` - Latency histogram (P95, P99)
- `http_request_size_bytes` - Request sizes
- `http_response_size_bytes` - Response sizes

**Checkout Metrics (`/metrics` on port 8081):**

- `grpc_requests_total{method, status_code}` - gRPC request count
- `grpc_request_duration_seconds` - gRPC latency
- `grpc_errors_total{method, error_code}` - Error tracking

**Dashboard:** Application Metrics - Frontend & Checkout

## Testing the Stack

### Generate Load

The `loadgenerator` service automatically sends traffic to the frontend.

Watch real-time metrics:

```powershell
# Prometheus query browser
http://localhost:9090/graph

# Example queries:
rate(http_requests_total{service="frontend"}[1m])
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
```

### Verify Scraping

```powershell
# Check Prometheus targets
http://localhost:9090/targets

# All should show "UP":
- cadvisor
- frontend
- checkoutservice
- node-exporter
```

## Instrumenting Additional Services

### For Go Services (Product Catalog, Shipping, etc.)

1. Add to `go.mod`:

```go
github.com/prometheus/client_golang v1.20.5
```

2. Create `metrics.go` (see `frontend/metrics.go` for HTTP or `checkoutservice/metrics.go` for gRPC)

3. Add metrics endpoint or interceptor to `main.go`

4. Update Prometheus scrape config in `monitoring/docker/prometheus.yml`

### For Node.js Services (Currency, Payment)

```javascript
npm install prom-client

const client = require('prom-client');
const register = new client.Registry();

// Collect default metrics
client.collectDefaultMetrics({ register });

// Expose /metrics
app.get('/metrics', async (req, res) => {
  res.set('Content-Type', register.contentType);
  res.end(await register.metrics());
});
```

### For Python Services (Email, Recommendation)

```python
pip install prometheus-client

from prometheus_client import Counter, Histogram, generate_latest

requests_total = Counter('http_requests_total', 'Total requests')
request_duration = Histogram('http_request_duration_seconds', 'Request duration')

@app.route('/metrics')
def metrics():
    return generate_latest()
```

### For Java Services (AdService)

Add to `pom.xml`:

```xml
<dependency>
    <groupId>io.micrometer</groupId>
    <artifactId>micrometer-registry-prometheus</artifactId>
</dependency>
```

Expose `/actuator/prometheus` endpoint.

### For .NET Services (CartService)

Add to `.csproj`:

```xml
<PackageReference Include="prometheus-net.AspNetCore" Version="8.0.0" />
```

In `Startup.cs`:

```csharp
app.UseMetricServer();
```

## BackTrack Integration Points

For your thesis on automated rollback:

### 1. Deployment Event Detection

Add annotation to Grafana:

```bash
curl -X POST http://localhost:3001/api/annotations \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Deployment: v1.2.3",
    "tags": ["deployment"],
    "time": '$(date +%s)000'
  }'
```

### 2. Anomaly Detection Queries

**Sudden error rate increase:**

```promql
rate(http_requests_total{status_code=~"5.."}[1m]) > 0.05
```

**Latency spike:**

```promql
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1.0
```

**Memory leak:**

```promql
deriv(container_memory_usage_bytes[10m]) > 10000000
```

### 3. Rollback Verification

After rollback, verify:

```promql
# Error rate back to normal
rate(http_requests_total{status_code=~"5.."}[5m]) < 0.01

# Latency recovered
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) < 0.5
```

## Prometheus Configuration Details

**Scrape Interval:** 15s (global default)  
**Retention:** 30 days  
**Storage:** Docker volume `prometheus-data`

**Key Jobs:**

- `cadvisor` - Container metrics (10s interval for more responsiveness)
- `frontend` - HTTP metrics
- `checkoutservice` - gRPC metrics (port 8081 for separation)
- `node-exporter` - Host metrics

## Grafana Dashboard Tips

### Creating Custom Panels

1. Click **Add panel** → **Add new panel**
2. Enter PromQL query
3. Choose visualization (Time series, Gauge, Bar chart)
4. Set thresholds for alerts

### Common PromQL Patterns

**Rate (requests/sec):**

```promql
rate(http_requests_total[1m])
```

**Percentage:**

```promql
sum(rate(http_requests_total{status_code=~"5.."}[5m]))
/
sum(rate(http_requests_total[5m]))
```

**Aggregation:**

```promql
sum by (service) (container_memory_usage_bytes)
```

## Troubleshooting

### Prometheus Not Scraping

```powershell
# Check Prometheus logs
docker logs prometheus

# Verify network connectivity
docker exec prometheus wget -O- http://frontend:8080/metrics
```

### Grafana Shows "No Data"

1. Check datasource: **Configuration** → **Data sources** → **Prometheus**
2. Test connection (should show green checkmark)
3. Verify Prometheus has data: `http://localhost:9090/graph`

### Services Not Exposing Metrics

```powershell
# Check if endpoint is accessible
curl http://localhost:8080/metrics  # frontend
curl http://localhost:8081/metrics  # checkoutservice (on separate port)

# Check service logs
docker logs frontend
```

### cAdvisor Not Running on Windows

If cAdvisor fails to start, remove these from `docker-compose.monitoring.yml`:

```yaml
privileged: true
devices:
  - /dev/kmsg
```

## Cleanup

```powershell
# Stop monitoring stack
docker-compose -f docker-compose.monitoring.yml down

# Remove volumes (deletes all metrics data)
docker-compose -f docker-compose.monitoring.yml down -v
```

## Next Steps

1. **Instrument remaining services** - Start with high-traffic services
2. **Add alerting** - Configure Alertmanager for Prometheus
3. **Create business metrics** - Track cart abandonment, checkout success rate
4. **Integrate with logs** - Add Loki for log aggregation
5. **Build BackTrack algorithm** - Use Prometheus API to fetch metrics programmatically

## API Access for BackTrack

### Query Prometheus from Code

```python
import requests

response = requests.get(
    'http://localhost:9090/api/v1/query',
    params={'query': 'rate(http_requests_total[1m])'}
)
metrics = response.json()
```

### Grafana API

```bash
# Export dashboard
curl -H "Authorization: Bearer <api-key>" \
  http://localhost:3001/api/dashboards/uid/container-overview
```

## Resources

- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3001`
- cAdvisor: `http://localhost:8080`
- Frontend Metrics: `http://localhost:8080/metrics`
- Checkout Metrics: `http://localhost:8081/metrics`

## Key Files Created

```
docker-compose.monitoring.yml           # Main monitoring stack
monitoring/docker/prometheus.yml        # Prometheus scrape config
monitoring/docker/grafana/
  provisioning/
    datasources/prometheus.yml          # Auto-configure Prometheus
    dashboards/dashboards.yml           # Auto-load dashboards
  dashboards/
    container-metrics.json              # Container dashboard
    application-metrics.json            # App metrics dashboard
src/frontend/metrics.go                 # Frontend instrumentation
src/checkoutservice/metrics.go          # Checkout instrumentation
```

---

**Your monitoring stack is now production-ready for DevOps thesis work!**
