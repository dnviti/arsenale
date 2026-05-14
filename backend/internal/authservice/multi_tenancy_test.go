package authservice

import (
	"context"
	"testing"
)

func TestNormalizeLoginUserForRuntimeKeepsAllMembershipsWhenDisabled(t *testing.T) {
	t.Parallel()

	user := loginUser{
		ID: "user-1",
		Memberships: []loginMembership{
			{TenantID: "tenant-a", Status: "ACCEPTED", IsActive: true},
			{TenantID: "tenant-b", Status: "ACCEPTED"},
			{TenantID: "tenant-c", Status: "PENDING"},
		},
	}

	got, err := (Service{}).normalizeLoginUserForRuntime(context.Background(), user)
	if err != nil {
		t.Fatalf("normalizeLoginUserForRuntime() error = %v", err)
	}
	if len(got.Memberships) != len(user.Memberships) {
		t.Fatalf("expected all memberships, got %#v", got.Memberships)
	}
}
