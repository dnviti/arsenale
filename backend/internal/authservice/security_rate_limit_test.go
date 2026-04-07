package authservice

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestEnforceLoginMFARateLimitBlocksAfterFiveAttempts(t *testing.T) {
	t.Parallel()

	svc := newRateLimitedAuthService(t)
	ctx := context.Background()

	for i := 0; i < loginMFARateLimitMaxAttempts; i++ {
		if err := svc.enforceLoginMFARateLimit(ctx, "user-1", "203.0.113.10"); err != nil {
			t.Fatalf("attempt %d error = %v", i+1, err)
		}
	}

	err := svc.enforceLoginMFARateLimit(ctx, "user-1", "203.0.113.10")
	var reqErr *requestError
	if !errors.As(err, &reqErr) {
		t.Fatalf("expected requestError, got %v", err)
	}
	if reqErr.status != 429 {
		t.Fatalf("status = %d, want 429", reqErr.status)
	}
}

func TestEnforceLoginRateLimitBypassesWhitelistedPrivateIP(t *testing.T) {
	t.Setenv("RATE_LIMIT_WHITELIST_CIDRS", "10.0.0.0/8")

	svc := newRateLimitedAuthService(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		if err := svc.enforceLoginRateLimit(ctx, "10.89.5.16"); err != nil {
			t.Fatalf("attempt %d error = %v", i+1, err)
		}
	}
}

func TestEnforceLoginRateLimitHonorsExplicitlyDisabledWhitelist(t *testing.T) {
	t.Setenv("RATE_LIMIT_WHITELIST_CIDRS", "")
	t.Setenv("LOGIN_RATE_LIMIT_MAX_ATTEMPTS", "2")

	svc := newRateLimitedAuthService(t)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if err := svc.enforceLoginRateLimit(ctx, "10.89.5.16"); err != nil {
			t.Fatalf("attempt %d error = %v", i+1, err)
		}
	}

	err := svc.enforceLoginRateLimit(ctx, "10.89.5.16")
	var reqErr *requestError
	if !errors.As(err, &reqErr) {
		t.Fatalf("expected requestError, got %v", err)
	}
	if reqErr.status != 429 {
		t.Fatalf("status = %d, want 429", reqErr.status)
	}
}

func TestEnforceLoginMFARateLimitBypassesWhitelistedPrivateIP(t *testing.T) {
	t.Setenv("RATE_LIMIT_WHITELIST_CIDRS", "10.0.0.0/8")

	svc := newRateLimitedAuthService(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		if err := svc.enforceLoginMFARateLimit(ctx, "user-1", "10.89.5.16"); err != nil {
			t.Fatalf("attempt %d error = %v", i+1, err)
		}
	}
}

func newRateLimitedAuthService(t *testing.T) Service {
	t.Helper()

	server, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	t.Cleanup(server.Close)

	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return Service{Redis: client}
}
