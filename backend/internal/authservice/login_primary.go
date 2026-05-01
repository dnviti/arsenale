package authservice

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func (s Service) Login(ctx context.Context, email, password, ipAddress, userAgent string) (loginFlow, error) {
	if s.DB == nil {
		return loginFlow{}, fmt.Errorf("postgres is not configured")
	}
	if len(s.JWTSecret) == 0 {
		return loginFlow{}, fmt.Errorf("JWT secret is not configured")
	}
	if email == "" || password == "" {
		return loginFlow{}, &requestError{status: 400, message: "Email and password are required"}
	}
	if err := s.enforceLoginRateLimit(ctx, ipAddress); err != nil {
		return loginFlow{}, err
	}
	if ldapFlow, err := s.tryLDAPLogin(ctx, email, password, ipAddress, userAgent); err != nil {
		return loginFlow{}, err
	} else if ldapFlow != nil {
		return *ldapFlow, nil
	}

	user, err := s.loadLoginUser(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = s.insertStandaloneAuditLog(ctx, nil, "LOGIN_FAILURE", map[string]any{
				"reason": "user_not_found",
				"email":  email,
			}, ipAddress)
			return loginFlow{}, &requestError{status: 401, message: "Invalid email or password"}
		}
		return loginFlow{}, err
	}

	allowlistDecision := evaluateIPAllowlist(user.ActiveTenant, ipAddress)
	if allowlistDecision.Blocked {
		return loginFlow{}, s.rejectBlockedIPAllowlist(ctx, user.ID, ipAddress)
	}

	if !user.Enabled {
		_ = s.insertStandaloneAuditLog(ctx, &user.ID, "LOGIN_FAILURE", map[string]any{
			"reason": "account_disabled",
			"email":  email,
		}, ipAddress)
		return loginFlow{}, &requestError{status: 403, message: "Your account has been disabled. Contact your administrator."}
	}

	effectiveThreshold := s.effectiveLockoutThreshold(user.ActiveTenant)
	effectiveDuration := s.effectiveLockoutDuration(user.ActiveTenant)

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		remainingMin := int(time.Until(*user.LockedUntil).Round(time.Minute).Minutes())
		if remainingMin < 1 {
			remainingMin = 1
		}
		_ = s.insertStandaloneAuditLog(ctx, &user.ID, "LOGIN_FAILURE", map[string]any{
			"reason": "account_locked",
			"email":  email,
		}, ipAddress)
		return loginFlow{}, &requestError{status: http.StatusLocked, message: fmt.Sprintf("Account is temporarily locked. Try again in %d minute%s.", remainingMin, plural(remainingMin))}
	}

	if user.PasswordHash == nil || *user.PasswordHash == "" {
		_ = s.insertStandaloneAuditLog(ctx, &user.ID, "LOGIN_FAILURE", map[string]any{
			"reason": "oauth_only_account",
			"email":  email,
		}, ipAddress)
		return loginFlow{}, &requestError{status: 400, message: "This account uses social login. Please sign in with your OAuth provider."}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		if err := s.recordInvalidPassword(ctx, user.ID, email, ipAddress, user.FailedLoginAttempts, effectiveThreshold, effectiveDuration); err != nil {
			return loginFlow{}, err
		}
		return loginFlow{}, &requestError{status: 401, message: "Invalid email or password"}
	}

	if s.EmailVerify && !user.EmailVerified {
		_ = s.insertStandaloneAuditLog(ctx, &user.ID, "LOGIN_FAILURE", map[string]any{
			"reason": "email_not_verified",
			"email":  email,
		}, ipAddress)
		return loginFlow{}, &requestError{status: 403, message: "Email not verified. Please check your inbox or resend the verification email."}
	}

	if err := s.resetLoginCounters(ctx, user.ID, user.FailedLoginAttempts, user.LockedUntil); err != nil {
		return loginFlow{}, err
	}

	if !user.WebAuthnEnabled {
		if err := s.storeVaultSession(ctx, user.ID, password, user); err != nil {
			return loginFlow{}, err
		}
	}

	return s.finalizePrimaryLogin(ctx, user, primaryMethodPassword, ipAddress, userAgent)
}

func (s Service) effectiveLockoutThreshold(active *loginMembership) int {
	if active != nil && active.AccountLockoutThreshold != nil {
		return *active.AccountLockoutThreshold
	}
	if value := os.Getenv("ACCOUNT_LOCKOUT_THRESHOLD"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return 10
}

func (s Service) effectiveLockoutDuration(active *loginMembership) time.Duration {
	if active != nil && active.AccountLockoutDurationMs != nil {
		return time.Duration(*active.AccountLockoutDurationMs) * time.Millisecond
	}
	if value := os.Getenv("ACCOUNT_LOCKOUT_DURATION_MS"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return time.Duration(parsed) * time.Millisecond
		}
	}
	return 30 * time.Minute
}

func (s Service) recordInvalidPassword(ctx context.Context, userID, email, ipAddress string, failedAttempts, threshold int, duration time.Duration) error {
	newAttempts := failedAttempts + 1
	var lockedUntil any = nil
	var storedAttempts = newAttempts
	accountLocked := false
	if newAttempts >= threshold {
		lockTime := time.Now().Add(duration)
		lockedUntil = lockTime
		storedAttempts = 0
		accountLocked = true
	}

	if _, err := s.DB.Exec(
		ctx,
		`UPDATE "User"
		    SET "failedLoginAttempts" = $2,
		        "lockedUntil" = $3
		  WHERE id = $1`,
		userID,
		storedAttempts,
		lockedUntil,
	); err != nil {
		return fmt.Errorf("update failed login attempts: %w", err)
	}

	return s.insertStandaloneAuditLog(ctx, &userID, "LOGIN_FAILURE", map[string]any{
		"reason":         "invalid_password",
		"email":          email,
		"failedAttempts": newAttempts,
		"accountLocked":  accountLocked,
	}, ipAddress)
}

func (s Service) resetLoginCounters(ctx context.Context, userID string, failedAttempts int, lockedUntil *time.Time) error {
	if failedAttempts == 0 && lockedUntil == nil {
		return nil
	}
	if _, err := s.DB.Exec(
		ctx,
		`UPDATE "User"
		    SET "failedLoginAttempts" = 0,
		        "lockedUntil" = NULL
		  WHERE id = $1`,
		userID,
	); err != nil {
		return fmt.Errorf("reset failed login counters: %w", err)
	}
	return nil
}
