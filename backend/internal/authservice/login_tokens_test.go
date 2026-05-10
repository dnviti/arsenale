package authservice

import (
	"testing"
	"time"
)

func TestBrowserRefreshTokenLifetimeUsesConfiguredTTL(t *testing.T) {
	now := time.Date(2026, time.May, 10, 12, 0, 0, 0, time.UTC)
	service := Service{RefreshCookieTTL: 30 * 24 * time.Hour}

	lifetime := service.browserRefreshTokenLifetime(loginUser{}, now)

	if lifetime.cookieTTL != 30*24*time.Hour {
		t.Fatalf("cookieTTL = %s, want 720h", lifetime.cookieTTL)
	}
	if !lifetime.expiresAt.Equal(now.Add(30 * 24 * time.Hour)) {
		t.Fatalf("expiresAt = %s, want %s", lifetime.expiresAt, now.Add(30*24*time.Hour))
	}
}

func TestCLIRefreshTokenLifetimeIsPersistent(t *testing.T) {
	lifetime := cliRefreshTokenLifetime()

	if !lifetime.expiresAt.Equal(persistentRefreshTokenExpiresAt) {
		t.Fatalf("expiresAt = %s, want %s", lifetime.expiresAt, persistentRefreshTokenExpiresAt)
	}
	if lifetime.cookieTTL != 0 {
		t.Fatalf("cookieTTL = %s, want 0", lifetime.cookieTTL)
	}
}

func TestCLIRefreshTokenFamilyIsTagged(t *testing.T) {
	family := newCLIRefreshTokenFamily()

	if !isCLIRefreshTokenFamily(family) {
		t.Fatalf("expected %q to be treated as a CLI refresh token family", family)
	}
	if isCLIRefreshTokenFamily("browser-family") {
		t.Fatal("browser token family was treated as a CLI refresh token family")
	}
}
