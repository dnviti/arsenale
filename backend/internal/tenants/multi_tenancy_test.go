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

func TestFilterTenantMembershipsForRuntimePrefersActiveAccepted(t *testing.T) {
	t.Parallel()

	filtered := filterTenantMembershipsForRuntime([]tenantMembershipResponse{
		{TenantID: "pending", Status: "PENDING"},
		{TenantID: "active", Status: "ACCEPTED", IsActive: true},
		{TenantID: "accepted", Status: "ACCEPTED"},
	}, false)
	if len(filtered) != 1 || filtered[0].TenantID != "active" {
		t.Fatalf("expected active membership, got %#v", filtered)
	}
}

func TestFilterTenantMembershipsForRuntimeFallsBackToAcceptedThenFirst(t *testing.T) {
	t.Parallel()

	accepted := filterTenantMembershipsForRuntime([]tenantMembershipResponse{
		{TenantID: "pending", Status: "PENDING"},
		{TenantID: "accepted", Status: "ACCEPTED"},
	}, false)
	if len(accepted) != 1 || accepted[0].TenantID != "accepted" {
		t.Fatalf("expected accepted membership, got %#v", accepted)
	}

	first := filterTenantMembershipsForRuntime([]tenantMembershipResponse{
		{TenantID: "pending-a", Status: "PENDING"},
		{TenantID: "pending-b", Status: "PENDING"},
	}, false)
	if len(first) != 1 || first[0].TenantID != "pending-a" {
		t.Fatalf("expected first membership, got %#v", first)
	}
}
