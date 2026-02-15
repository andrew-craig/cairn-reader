# Metrics

**Priority:** P2
**Status:** pending
**Task ID:** Pre-Go Live

## Problem

Before going live, comprehensive metrics must be collected and exposed for monitoring system health, performance, and business KPIs.

## Impact

Without metrics:
- No visibility into system performance
- Cannot identify bottlenecks or performance regressions
- Difficult to capacity plan
- No data-driven optimization decisions
- Cannot track business KPIs

## Proposed Solution

Implement comprehensive metrics collection:

1. **Infrastructure metrics:**
   - HTTP request latency (P50, P95, P99)
   - Error rates by endpoint and error type
   - Request throughput (RPS)
   - Database query latency and counts
   - Database connection pool usage
   - Cache hit/miss ratios
   - JWT validation latency
   - Vault key fetch latency

2. **Business metrics:**
   - User registrations per day
   - Article recommendations per user
   - Voting activity
   - Feed subscription count
   - Content ingestion rate

3. **System health metrics:**
   - Memory usage
   - CPU usage
   - Disk usage
   - Database connections
   - Open file descriptors

4. **Implementation:**
   - Use Prometheus client libraries
   - Export metrics on `/metrics` endpoint
   - Set up Prometheus scraper
   - Create Grafana dashboards for visualization
   - Configure data retention (e.g., 30 days)

## Example metrics to add:

```go
// HTTP request metrics
http_requests_total{method,path,status}
http_request_duration_seconds{method,path,quantile}
http_errors_total{method,path,error_type}

// Database metrics
db_query_duration_seconds{operation,table}
db_connections{state}

// Business metrics
users_registered_total
articles_recommended_total
votes_cast_total

// Authentication metrics
jwt_validations_total{status}
jwt_validation_duration_seconds
```

## Files to Modify

- All service main files - add metrics initialization
- HTTP middleware - record request metrics
- Database operations - record query metrics
- `infrastructure/docker/dev/docker-compose.yml` - add Prometheus and Grafana
- Create Prometheus configuration

## Testing

- Verify metrics are being collected
- Check metric names and labels are consistent
- Validate dashboard displays data correctly
- Test metrics under load

## Related Tasks

- P2-Alerting
- P2-Load-Testing
