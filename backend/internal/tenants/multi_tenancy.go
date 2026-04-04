package tenants

func filterTenantMembershipsForRuntime(memberships []tenantMembershipResponse, multiTenancyEnabled bool) []tenantMembershipResponse {
	if multiTenancyEnabled || len(memberships) <= 1 {
		return memberships
	}
	for _, membership := range memberships {
		if membership.IsActive && membership.Status == "ACCEPTED" {
			return []tenantMembershipResponse{membership}
		}
	}
	for _, membership := range memberships {
		if membership.Status == "ACCEPTED" {
			return []tenantMembershipResponse{membership}
		}
	}
	return []tenantMembershipResponse{memberships[0]}
}
