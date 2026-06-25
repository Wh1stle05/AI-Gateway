package ratelimit

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestLimiterAllowsThenBlocks(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	limiter := New(client, Config{RequestsPerMinute: 6000, Burst: 2})

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		ok, err := limiter.Allow(ctx, "user-a")
		if err != nil {
			t.Fatalf("Allow() error = %v", err)
		}
		if !ok {
			t.Fatalf("request %d should be allowed", i)
		}
	}

	ok, err := limiter.Allow(ctx, "user-a")
	if err != nil {
		t.Fatalf("Allow() error = %v", err)
	}
	if ok {
		t.Fatal("third request should be blocked by burst limit")
	}
}

func TestLimiterIsolatedByKey(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	limiter := New(client, Config{RequestsPerMinute: 6000, Burst: 1})

	ctx := context.Background()
	ok, err := limiter.Allow(ctx, "user-a")
	if err != nil || !ok {
		t.Fatalf("user-a first request: ok=%v err=%v", ok, err)
	}

	ok, err = limiter.Allow(ctx, "user-b")
	if err != nil || !ok {
		t.Fatalf("user-b should have separate bucket: ok=%v err=%v", ok, err)
	}
}
