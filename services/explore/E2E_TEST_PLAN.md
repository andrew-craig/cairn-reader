# End-to-End Test Plan for Cairn Explore

## Test Objective
Validate the complete integration of the Fetcher and Recommender services, ensuring all core functionality works correctly from feed fetching to article recommendations.

## Architecture Under Test

```
Kagi Feed List → Fetcher DB ← Fetcher Service (8080) → Recommender Service (8081) → Recommender DB
                      ↓                                           ↓
                Feed Management                          Article Storage & Recommendations
```

## Test Scenarios

### 1. Service Infrastructure Tests
**Objective**: Verify all services start correctly and are healthy

- [ ] 1.1. Start Docker Compose stack (postgres DBs, fetcher, recommender)
- [ ] 1.2. Verify fetcher health endpoint (`GET /health`)
- [ ] 1.3. Verify recommender health endpoint (`GET /health`)
- [ ] 1.4. Check database connectivity for both services
- [ ] 1.5. Verify Docker containers are running without errors

**Expected Results**:
- All containers running and healthy
- Health endpoints return `{"status":"healthy"}`
- No connection errors in logs

---

### 2. Feed Management Tests
**Objective**: Validate feed sync and storage in fetcher database

- [ ] 2.1. Verify feed sync from Kagi Small Web collection
- [ ] 2.2. Check feeds are stored in fetcher database
- [ ] 2.3. Verify feeds table schema (url, enabled, last_fetched_at, consecutive_failures)
- [ ] 2.4. Confirm initial state: enabled=true, consecutive_failures=0
- [ ] 2.5. Verify no duplicate feed URLs

**Expected Results**:
- Feeds successfully synced and stored
- All feeds enabled by default
- Unique URLs enforced

**Validation Commands**:
```sql
SELECT COUNT(*) FROM feeds;
SELECT * FROM feeds WHERE enabled = false;
SELECT url, COUNT(*) FROM feeds GROUP BY url HAVING COUNT(*) > 1;
```

---

### 3. Feed Fetching Tests
**Objective**: Validate one-feed-per-minute fetching logic

- [ ] 3.1. Trigger manual fetch (`POST /fetch` or wait for automatic fetch)
- [ ] 3.2. Verify only 1 feed fetched per minute
- [ ] 3.3. Confirm feed prioritization (never-fetched first, then oldest last_fetched_at)
- [ ] 3.4. Verify last_fetched_at timestamp updated
- [ ] 3.5. Check fetch_history table for fetch records
- [ ] 3.6. Monitor consecutive_failures counter on fetch errors

**Expected Results**:
- Exactly 1 feed processed per minute
- Never-fetched feeds prioritized
- Timestamps updated correctly
- Fetch history recorded

**Validation Commands**:
```sql
SELECT * FROM feeds WHERE last_fetched_at IS NOT NULL ORDER BY last_fetched_at DESC LIMIT 5;
SELECT * FROM fetch_history ORDER BY fetched_at DESC LIMIT 10;
```

---

### 4. Article Submission Tests
**Objective**: Verify articles are sent from Fetcher to Recommender

- [ ] 4.1. Verify fetcher submits articles via `POST /api/v1/articles`
- [ ] 4.2. Check article deduplication (SHA256 hash IDs)
- [ ] 4.3. Verify article metadata stored (title, link, content, published_at, categories)
- [ ] 4.4. Confirm feed_id reference set correctly
- [ ] 4.5. Verify batch submission handling

**Expected Results**:
- Articles successfully submitted to recommender
- Duplicate articles handled with ON CONFLICT
- All metadata fields populated
- feed_id links back to fetcher database feed ID

**Validation Commands**:
```sql
SELECT COUNT(*) FROM articles;
SELECT id, title, feed_id, published_at FROM articles ORDER BY created_at DESC LIMIT 10;
SELECT COUNT(*) as article_count, feed_id FROM articles GROUP BY feed_id;
```

---

### 5. Recommendation Engine Tests
**Objective**: Validate recommendation algorithm and user tracking

- [ ] 5.1. Request recommendations for new user (`GET /api/v1/recommendations/{userID}`)
- [ ] 5.2. Verify 5 articles returned
- [ ] 5.3. Check recommendation scoring (recency, length, title, randomization)
- [ ] 5.4. Request recommendations for same user again
- [ ] 5.5. Verify different articles recommended (randomization working)

**Expected Results**:
- Exactly 5 articles per request
- Articles sorted by score
- Variety in recommendations across requests

**Test Commands**:
```bash
curl http://localhost:8081/api/v1/recommendations/test-user-001
curl http://localhost:8081/api/v1/recommendations/test-user-002
```

---

### 6. User Interaction Tests
**Objective**: Test read status tracking and filtering

- [ ] 6.1. Mark article as read (`POST /api/v1/articles/read`)
- [ ] 6.2. Verify user_articles table entry created
- [ ] 6.3. Request recommendations for same user
- [ ] 6.4. Confirm read article NOT in recommendations
- [ ] 6.5. Mark multiple articles as read
- [ ] 6.6. Verify all read articles filtered from recommendations

**Expected Results**:
- Read status persisted correctly
- Read articles excluded from future recommendations
- User tracking works independently per user

**Test Commands**:
```bash
# Get recommendations
RECS=$(curl -s http://localhost:8081/api/v1/recommendations/test-user-001)
ARTICLE_ID=$(echo $RECS | jq -r '.articles[0].id')

# Mark as read
curl -X POST http://localhost:8081/api/v1/articles/read \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":\"test-user-001\",\"article_id\":\"$ARTICLE_ID\"}"

# Verify filtered out
curl http://localhost:8081/api/v1/recommendations/test-user-001 | jq
```

**Validation SQL**:
```sql
SELECT * FROM user_articles WHERE user_id = 'test-user-001';
```

---

### 7. Feed Health Tracking Tests
**Objective**: Validate automatic feed disabling on failures

- [ ] 7.1. Identify feed with connection errors
- [ ] 7.2. Verify consecutive_failures counter increments
- [ ] 7.3. Simulate 10 consecutive failures
- [ ] 7.4. Verify feed auto-disabled (enabled = false)
- [ ] 7.5. Confirm disabled feeds not fetched
- [ ] 7.6. Simulate successful fetch
- [ ] 7.7. Verify consecutive_failures reset to 0

**Expected Results**:
- Failure counter increments on errors
- Feed disabled after 10 failures
- Disabled feeds skipped in fetch cycle
- Counter resets on successful fetch

**Validation Commands**:
```sql
SELECT url, enabled, consecutive_failures, last_fetched_at
FROM feeds
WHERE consecutive_failures > 0
ORDER BY consecutive_failures DESC
LIMIT 10;

SELECT COUNT(*) FROM feeds WHERE enabled = false;
```

---

### 8. Multi-User Isolation Tests
**Objective**: Verify user data isolation

- [ ] 8.1. Create multiple test users
- [ ] 8.2. Mark different articles as read for each user
- [ ] 8.3. Verify each user gets personalized recommendations
- [ ] 8.4. Confirm no cross-user data leakage

**Expected Results**:
- User recommendations independent
- Read status isolated per user
- No data leakage between users

**Test Users**: test-user-001, test-user-002, test-user-003

---

### 9. Performance & Load Tests
**Objective**: Validate system performance under normal conditions

- [ ] 9.1. Monitor fetcher memory usage during fetch cycle
- [ ] 9.2. Check recommender response times (should be < 100ms)
- [ ] 9.3. Verify batch article submission performance
- [ ] 9.4. Monitor database connection pooling
- [ ] 9.5. Check for memory leaks over 10-minute period

**Expected Results**:
- Stable memory usage
- Fast recommendation response times
- No connection pool exhaustion

---

### 10. Error Handling Tests
**Objective**: Validate graceful error handling

- [ ] 10.1. Submit invalid article data to recommender
- [ ] 10.2. Request recommendations for user with no available articles
- [ ] 10.3. Mark non-existent article as read
- [ ] 10.4. Test fetcher behavior with unreachable feeds
- [ ] 10.5. Verify appropriate HTTP status codes (400, 404, 500)

**Expected Results**:
- Graceful error responses
- No service crashes
- Proper HTTP status codes
- Error messages in logs

---

## Test Execution Checklist

### Prerequisites
- [ ] Docker and Docker Compose installed
- [ ] All containers stopped: `docker-compose down`
- [ ] Clean database volumes: `docker volume rm cairn-explore_postgres_data cairn-explore_fetcher_postgres_data`

### Execution Steps
1. Start services: `docker-compose up --build -d`
2. Wait 30 seconds for initialization
3. Run each test scenario in order
4. Collect logs: `docker-compose logs > e2e-test-logs.txt`
5. Document results

### Cleanup
```bash
docker-compose down
docker volume rm cairn-explore_postgres_data cairn-explore_fetcher_postgres_data
```

---

## Success Criteria

All tests must pass with:
- ✅ All services healthy
- ✅ Feeds successfully synced and fetched
- ✅ Articles submitted and stored correctly
- ✅ Recommendations returned with correct filtering
- ✅ User tracking working independently
- ✅ Feed health tracking operational
- ✅ No critical errors in logs
- ✅ Response times acceptable (< 100ms for recommendations)

---

## Known Limitations (Current Implementation)

Based on CLAUDE.md, the following features are **NOT YET IMPLEMENTED**:
- ❌ Voting system (upvote/downvote endpoints)
- ❌ Quality score algorithm: `(upvotes + (downvotes * 3)) / recommends`
- ❌ Enhanced recommendation algorithm (4 high-quality + 1 low-exposure)
- ❌ Article cleanup (90-day retention)
- ❌ Votes and recommendations tables

These will be tested after implementation per RECOMMENDER_PLAN.md.

---

## Test Environment

**Services**:
- Fetcher: `http://localhost:8080`
- Recommender: `http://localhost:8081`
- Fetcher DB: PostgreSQL on port 5433
- Recommender DB: PostgreSQL on port 5432

**Docker Compose**: See `docker-compose.yml`

**Test Duration**: Approximately 15-20 minutes for full test suite
