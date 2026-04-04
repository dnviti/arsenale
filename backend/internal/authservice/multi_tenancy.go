package authservice

import (
	"context"
	"fmt"
)

func (s Service) normalizeLoginUserForRuntime(ctx context.Context, user loginUser) (loginUser, error) {
	if s.Features.MultiTenancyEnabled {
		return user, nil
	}

	selected, ok := selectSingleTenantMembership(user.Memberships)
	if !ok {
		user.ActiveTenant = nil
		user.Memberships = nil
		return user, nil
	}

	if selected.Status == "ACCEPTED" {
		if _, err := s.DB.Exec(ctx, `
UPDATE "TenantMember"
SET "isActive" = false
WHERE "userId" = $1
  AND "isActive" = true
  AND "tenantId" <> $2
`, user.ID, selected.TenantID); err != nil {
			return loginUser{}, fmt.Errorf("deactivate extra tenant memberships: %w", err)
		}
		if !selected.IsActive {
			if _, err := s.DB.Exec(ctx, `
UPDATE "TenantMember"
SET status = 'ACCEPTED', "isActive" = true
WHERE "tenantId" = $1 AND "userId" = $2
`, selected.TenantID, user.ID); err != nil {
				return loginUser{}, fmt.Errorf("activate single-tenant membership: %w", err)
			}
			selected.IsActive = true
		}
		active := selected
		user.ActiveTenant = &active
	} else {
		user.ActiveTenant = nil
	}

	user.Memberships = []loginMembership{selected}
	return user, nil
}

func selectSingleTenantMembership(memberships []loginMembership) (loginMembership, bool) {
	if len(memberships) == 0 {
		return loginMembership{}, false
	}
	for _, membership := range memberships {
		if membership.IsActive && membership.Status == "ACCEPTED" {
			return membership, true
		}
	}
	for _, membership := range memberships {
		if membership.Status == "ACCEPTED" {
			return membership, true
		}
	}
	return memberships[0], true
}
