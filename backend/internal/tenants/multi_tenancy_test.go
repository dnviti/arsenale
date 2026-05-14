package tenants

import "testing"

func TestFilterTenantMembershipsForRuntimeReturnsAllWhenEnabled(t *testing.T) {
	t.Parallel()

	items := []tenantMembershipResponse{
		{TenantID: "one", Status: "ACCEPTED", IsActive: true},
		{TenantID: "two", Status: "ACCEPTED"},
	}
	filtered := filterTenantMembershipsForRuntime(items, true)
	if len(filtered) != 2 {
		t.Fatalf("expected all memberships, got %d", len(filtered))
	}
}

func TestFilterTenantMembershipsForRuntimeReturnsAllWhenDisabled(t *testing.T) {
	t.Parallel()

	items := []tenantMembershipResponse{
		{TenantID: "pending", Status: "PENDING"},
		{TenantID: "active", Status: "ACCEPTED", IsActive: true},
		{TenantID: "accepted", Status: "ACCEPTED"},
	}
	filtered := filterTenantMembershipsForRuntime(items, false)
	if len(filtered) != len(items) {
		t.Fatalf("expected all memberships, got %#v", filtered)
	}
}
