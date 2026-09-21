package handler

import (
	"context"

	identityv1 "taskmanager/common/gen/go/identity/v1"
	"taskmanager/service/identity/internal/dependency"
)

func (h *ProfileHandler) LoginWithProvider(ctx context.Context, req *identityv1.LoginWithProviderRequest) (*identityv1.LoginWithProviderResponse, error) {
	if err := dependency.ValidateLoginWithProvider(req); err != nil {
		return nil, err
	}

	profile, created, err := h.auth.LoginWithProvider(ctx, dependency.ProviderLoginFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}

	return &identityv1.LoginWithProviderResponse{
		Profile: dependency.ProfileToProto(profile),
		Created: created,
	}, nil
}
