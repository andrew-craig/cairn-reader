package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// RedisScripter is the subset of the Redis client interface needed for
// the sliding-window rate limiter. *github.com/redis/go-redis/v9.Client
// satisfies this interface. It is an interface so tests can inject a mock
// without a real Redis server.
type RedisScripter interface {
	// EvalInt64 runs a Lua script and returns an int64 result.
	// Implementations should map to EVAL on the Redis server.
	EvalInt64(ctx context.Context, script string, keys []string, args ...interface{}) (int64, error)
}

// slidingWindowScript is a Lua script for a sliding window rate limiter.
// Scores are millisecond timestamps; members are unique strings to avoid
// collisions from concurrent requests.
//
// KEYS[1]: sorted-set key
// ARGV[1]: current Unix timestamp in milliseconds
// ARGV[2]: window size in milliseconds
// ARGV[3]: request limit
// ARGV[4]: unique member id for this request
//
// Returns 1 if the request is allowed, 0 if it should be denied.
const slidingWindowScript = `
local key    = KEYS[1]
local now    = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit  = tonumber(ARGV[3])
local member = ARGV[4]
local cutoff = now - window

redis.call('ZREMRANGEBYSCORE', key, '-inf', cutoff)

local count = redis.call('ZCARD', key)
if count < limit then
    redis.call('ZADD', key, now, member)
    redis.call('PEXPIRE', key, window)
    return 1
end
return 0
`

// redisRateLimiter implements a sliding-window rate limiter backed by Redis.
// It is safe for multi-instance deployments.
type redisRateLimiter struct {
	client RedisScripter
	limit  int
	window time.Duration
}

// NewRedisRateLimiter creates a Redis-backed sliding-window rate limiter.
//   - client: Redis client implementing RedisScripter
//   - limit: maximum requests allowed per window
//   - window: the duration of the sliding window
func NewRedisRateLimiter(client RedisScripter, limit int, window time.Duration) *redisRateLimiter {
	return &redisRateLimiter{client: client, limit: limit, window: window}
}

// allow returns true if the request identified by key is within the rate limit.
// On Redis errors it fails open (allows the request) to avoid blocking all traffic.
func (r *redisRateLimiter) allow(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var buf [8]byte
	_, _ = rand.Read(buf[:])
	member := fmt.Sprintf("%d:%s", time.Now().UnixMilli(), hex.EncodeToString(buf[:]))

	result, err := r.client.EvalInt64(ctx, slidingWindowScript,
		[]string{fmt.Sprintf("ratelimit:%s", key)},
		time.Now().UnixMilli(),
		r.window.Milliseconds(),
		r.limit,
		member,
	)
	if err != nil {
		// Fail open: don't block traffic when Redis is unavailable.
		return true
	}

	return result == 1
}
