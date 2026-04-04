package authservice

import "testing"

func TestSelectSingleTenantMembershipPrefersActiveAccepted(t *testing.T) {
	t.Parallel()

	selected, ok := selectSingleTenantMembership([]loginMembership{
		{TenantID: "pending", Status: "PENDING"},
		{TenantID: "accepted", Status: "ACCEPTED"},
		{TenantID: "active", Status: "ACCEPTED", IsActive: true},
	})
	if !ok {
		t.Fatal("expected a membership to be selected")
	}
	if selected.TenantID != "active" {
		t.Fatalf("expected active membership, got %q", selected.TenantID)
	}
}

func TestSelectSingleTenantMembershipFallsBackToFirstAccepted(t *testing.T) {
	t.Parallel()

	selected, ok := selectSingleTenantMembership([]loginMembership{
		{TenantID: "pending", Status: "PENDING"},
		{TenantID: "accepted", Status: "ACCEPTED"},
	})
	if !ok {
		t.Fatal("expected a membership to be selected")
	}
	if selected.TenantID != "accepted" {
		t.Fatalf("expected accepted membership, got %q", selected.TenantID)
	}
}

func TestSelectSingleTenantMembershipFallsBackToFirstMembership(t *testing.T) {
	t.Parallel()

	selected, ok := selectSingleTenantMembership([]loginMembership{
		{TenantID: "pending-a", Status: "PENDING"},
		{TenantID: "pending-b", Status: "PENDING"},
	})
	if !ok {
		t.Fatal("expected a membership to be selected")
	}
	if selected.TenantID != "pending-a" {
		t.Fatalf("expected first membership, got %q", selected.TenantID)
	}
}
