package circuitbreaker

import (
	"testing"
	"time"
)

func TestBreakerOpensAndRecovers(t *testing.T) {
	b := New(Config{
		FailureThreshold:    3,
		OpenTimeout:         20 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	})

	for i := 0; i < 3; i++ {
		if !b.Allow() {
			t.Fatalf("request %d: breaker should allow in closed state", i)
		}
		b.RecordFailure()
	}

	if b.Allow() {
		t.Fatal("breaker should reject while open")
	}
	if b.State() != "open" {
		t.Fatalf("state = %q, want open", b.State())
	}

	time.Sleep(25 * time.Millisecond)

	if !b.Allow() {
		t.Fatal("breaker should allow probe in half-open state")
	}
	b.RecordSuccess()
	if b.State() != "closed" {
		t.Fatalf("state = %q, want closed after success", b.State())
	}
}

func TestBreakerHalfOpenFailureReopens(t *testing.T) {
	b := New(Config{
		FailureThreshold:    1,
		OpenTimeout:         10 * time.Millisecond,
		HalfOpenMaxRequests: 2,
	})

	b.RecordFailure()
	time.Sleep(15 * time.Millisecond)

	if !b.Allow() {
		t.Fatal("expected half-open probe")
	}
	b.RecordFailure()

	if b.Allow() {
		t.Fatal("breaker should reject after half-open failure")
	}
	if b.State() != "open" {
		t.Fatalf("state = %q, want open", b.State())
	}
}
