package authservice

import "context"

func (s Service) normalizeLoginUserForRuntime(ctx context.Context, user loginUser) (loginUser, error) {
	return user, nil
}
