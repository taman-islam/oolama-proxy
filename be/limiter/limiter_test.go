package limiter_test

import (
	"lb/limiter"
	"testing"
	"time"
)

func TestCheckRPS_ExplicitUnlimited(t *testing.T) {
	lim := limiter.New()
	// Explicitly set INF_RPS to remove the rate limit.
	lim.SetLimits("user-a", limiter.INF_RPS, 0, 0)
	// With unlimited RPS, 100 rapid calls should all pass.
	for i := 0; i < 100; i++ {
		if err := lim.CheckRPS("user-a"); err != nil {
			t.Fatalf("expected no error on call %d, got: %v", i, err)
		}
	}
}

func TestCheckRPS_LimitEnforced(t *testing.T) {
	lim := limiter.New()
	lim.SetLimits("user-b", 2, 0, 0) // 2 RPS, no token quota, no per-request cap

	// First two calls should pass (burst = RPS).
	if err := lim.CheckRPS("user-b"); err != nil {
		t.Fatalf("call 1 should pass: %v", err)
	}
	if err := lim.CheckRPS("user-b"); err != nil {
		t.Fatalf("call 2 should pass: %v", err)
	}

	// Third call in the same instant should be rejected.
	if err := lim.CheckRPS("user-b"); err == nil {
		t.Fatal("call 3 should have been rate-limited")
	}

	// After 1 second, bucket refills — should pass again.
	time.Sleep(1100 * time.Millisecond)
	if err := lim.CheckRPS("user-b"); err != nil {
		t.Fatalf("call after refill should pass: %v", err)
	}
}

func TestCheckQuota_LimitEnforced(t *testing.T) {
	lim := limiter.New()
	lim.SetLimits("user-c", 0, 10, 0) // unlimited RPS, 10 token quota, no per-request cap

	if err := lim.CheckQuota("user-c"); err != nil {
		t.Fatalf("should pass before consuming tokens: %v", err)
	}

	// Consume exactly quota + grace; should now be rejected.
	lim.ConsumeTokens("user-c", 15) // 10 quota + 5 grace

	if err := lim.CheckQuota("user-c"); err == nil {
		t.Fatal("should be rejected after quota + grace exhausted")
	}
}

func TestConsumeTokens_Accumulates(t *testing.T) {
	lim := limiter.New()
	lim.SetLimits("user-d", 0, 100, 0)

	lim.ConsumeTokens("user-d", 30)
	lim.ConsumeTokens("user-d", 30)
	lim.ConsumeTokens("user-d", 30)

	// 90 used, 10 remaining + 5 grace — should still pass
	if err := lim.CheckQuota("user-d"); err != nil {
		t.Fatalf("90/100 used, should still pass: %v", err)
	}

	// Consume remaining quota + grace (10 + 5 = 15) to trigger rejection
	lim.ConsumeTokens("user-d", 15)

	// 105/100 — exceeds quota + grace, should fail
	if err := lim.CheckQuota("user-d"); err == nil {
		t.Fatal("105/100 used, should be rejected")
	}
}

func TestSetLimits_ResetsUsage(t *testing.T) {
	lim := limiter.New()
	lim.SetLimits("user-e", 0, 5, 0)
	lim.ConsumeTokens("user-e", 10) // consume quota (5) + grace (5)

	// Quota + grace exhausted — should be rejected
	if err := lim.CheckQuota("user-e"); err == nil {
		t.Fatal("should be rejected after quota + grace consumed")
	}

	// Admin resets limits — usage counter is reset
	lim.SetLimits("user-e", 0, 100, 0)
	if err := lim.CheckQuota("user-e"); err != nil {
		t.Fatalf("after limit reset, should pass: %v", err)
	}
}
