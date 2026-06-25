package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const tokenBucketScript = `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])

local data = redis.call('HMGET', key, 'tokens', 'last')
local tokens = tonumber(data[1])
local last = tonumber(data[2])

if tokens == nil then
  tokens = burst
  last = now
end

local elapsed = math.max(0, now - last)
tokens = math.min(burst, tokens + elapsed * rate / 1000)
last = now

if tokens < requested then
  redis.call('HMSET', key, 'tokens', tokens, 'last', last)
  redis.call('PEXPIRE', key, math.ceil(burst / rate * 2000))
  return 0
end

tokens = tokens - requested
redis.call('HMSET', key, 'tokens', tokens, 'last', last)
redis.call('PEXPIRE', key, math.ceil(burst / rate * 2000))
return 1
`

type Config struct {
	RequestsPerMinute int
	Burst             int
	KeyPrefix         string
}

type Limiter struct {
	client *redis.Client
	cfg    Config
	script *redis.Script
}

func New(client *redis.Client, cfg Config) *Limiter {
	if cfg.RequestsPerMinute <= 0 {
		cfg.RequestsPerMinute = 60
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 10
	}
	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "aigateway:rl:"
	}
	return &Limiter{
		client: client,
		cfg:    cfg,
		script: redis.NewScript(tokenBucketScript),
	}
}

func (l *Limiter) Ping(ctx context.Context) error {
	return l.client.Ping(ctx).Err()
}

func (l *Limiter) Allow(ctx context.Context, key string) (bool, error) {
	ratePerSec := float64(l.cfg.RequestsPerMinute) / 60.0
	now := time.Now().UnixMilli()

	result, err := l.script.Run(
		ctx,
		l.client,
		[]string{l.cfg.KeyPrefix + key},
		ratePerSec,
		l.cfg.Burst,
		now,
		1,
	).Int()
	if err != nil {
		return false, fmt.Errorf("rate limit script: %w", err)
	}
	return result == 1, nil
}

func (l *Limiter) RetryAfterSeconds() int {
	if l.cfg.RequestsPerMinute <= 0 {
		return 1
	}
	secs := 60 / l.cfg.RequestsPerMinute
	if secs < 1 {
		return 1
	}
	return secs
}
