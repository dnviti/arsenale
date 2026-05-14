package authservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRegisterCreatesUserWithGeneratedIDAndTimestamp(t *testing.T) {
	t.Setenv("SELF_SIGNUP_ENABLED", "true")
	t.Setenv("HIBP_FAIL_OPEN", "true")

	originalPasswordCheck := checkPasswordNotBreached
	checkPasswordNotBreached = func(context.Context, string) error { return nil }
	t.Cleanup(func() { checkPasswordNotBreached = originalPasswordCheck })

	db := &registerDBStub{}
	svc := Service{DB: db}

	result, err := svc.Register(context.Background(), "New.User@Example.com", "StrongPassword123!", "203.0.113.10")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if _, err := uuid.Parse(result.UserID); err != nil {
		t.Fatalf("UserID = %q, want generated UUID: %v", result.UserID, err)
	}
	if result.EmailVerifyRequired {
		t.Fatal("EmailVerifyRequired = true, want false")
	}
	if result.RecoveryKey == "" {
		t.Fatal("RecoveryKey is empty")
	}
	if !db.insertSawGeneratedID {
		t.Fatal("user insert did not pass a generated id")
	}
	if !db.insertSawUpdatedAt {
		t.Fatal(`user insert did not assign "updatedAt"`)
	}
}

type registerDBStub struct {
	insertSawGeneratedID bool
	insertSawUpdatedAt   bool
}

func (db *registerDBStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	switch {
	case strings.Contains(query, `FROM "AppConfig"`):
		return rowWithValues{"true"}
	case strings.Contains(query, `SELECT id FROM "User" WHERE email`):
		return rowWithError{pgx.ErrNoRows}
	case strings.Contains(query, `INSERT INTO "User"`):
		return db.insertUserRow(query, args...)
	default:
		return rowWithError{fmt.Errorf("unexpected query: %s", query)}
	}
}

func (db *registerDBStub) insertUserRow(query string, args ...any) pgx.Row {
	firstValueIsID := false
	if len(args) > 0 {
		if id, ok := args[0].(string); ok {
			_, err := uuid.Parse(id)
			firstValueIsID = err == nil
		}
	}
	db.insertSawGeneratedID = strings.Contains(query, `id,`) && firstValueIsID
	db.insertSawUpdatedAt = strings.Contains(query, `"updatedAt"`)

	if !db.insertSawGeneratedID {
		return rowWithError{errors.New(`ERROR: null value in column "id" of relation "User" violates not-null constraint (SQLSTATE 23502)`)}
	}
	if !db.insertSawUpdatedAt {
		return rowWithError{errors.New(`ERROR: null value in column "updatedAt" of relation "User" violates not-null constraint (SQLSTATE 23502)`)}
	}
	return rowWithValues{args[0].(string)}
}

func (db *registerDBStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (db *registerDBStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected Query")
}

func (db *registerDBStub) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return nil, errors.New("unexpected BeginTx")
}

type rowWithValues []any

func (r rowWithValues) Scan(dest ...any) error {
	if len(dest) != len(r) {
		return fmt.Errorf("scan destination count = %d, want %d", len(dest), len(r))
	}
	for i, value := range r {
		switch target := dest[i].(type) {
		case *string:
			*target = value.(string)
		default:
			return fmt.Errorf("unsupported scan destination %T", dest[i])
		}
	}
	return nil
}

type rowWithError struct {
	err error
}

func (r rowWithError) Scan(...any) error {
	return r.err
}
