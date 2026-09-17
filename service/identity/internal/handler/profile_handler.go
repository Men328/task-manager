package handler

import (
	"context"

	identityv1 "taskmanager/common/gen/go/identity/v1"
	"taskmanager/service/identity/internal/dependency"
	"taskmanager/service/identity/internal/service"
)

type ProfileHandler struct {
	identityv1.UnimplementedProfileServiceServer
	profiles service.ProfileService
}

func NewProfileHandler(profiles service.ProfileService) *ProfileHandler {
	return &ProfileHandler{profiles: profiles}
}

func (h *ProfileHandler) CreateProfile(ctx context.Context, req *identityv1.CreateProfileRequest) (*identityv1.CreateProfileResponse, error) {
	if err := dependency.ValidateCreateProfile(req); err != nil {
		return nil, err
	}

	created, err := h.profiles.Create(ctx, dependency.ProfileFromCreateRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &identityv1.CreateProfileResponse{Profile: dependency.ProfileToProto(created)}, nil
}

func (h *ProfileHandler) GetProfile(ctx context.Context, req *identityv1.GetProfileRequest) (*identityv1.GetProfileResponse, error) {
	if err := dependency.ValidateProfileID(req.GetId()); err != nil {
		return nil, err
	}

	p, err := h.profiles.Get(ctx, req.GetId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &identityv1.GetProfileResponse{Profile: dependency.ProfileToProto(p)}, nil
}

func (h *ProfileHandler) ListProfiles(ctx context.Context, req *identityv1.ListProfilesRequest) (*identityv1.ListProfilesResponse, error) {
	items, err := h.profiles.List(ctx, int(req.GetPageSize()))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}

	out := make([]*identityv1.Profile, 0, len(items))
	for _, p := range items {
		out = append(out, dependency.ProfileToProto(p))
	}
	return &identityv1.ListProfilesResponse{Profiles: out}, nil
}

func (h *ProfileHandler) UpdateProfile(ctx context.Context, req *identityv1.UpdateProfileRequest) (*identityv1.UpdateProfileResponse, error) {
	if err := dependency.ValidateProfileID(req.GetId()); err != nil {
		return nil, err
	}

	updated, err := h.profiles.Update(ctx, req.GetId(), dependency.ProfileUpdateFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &identityv1.UpdateProfileResponse{Profile: dependency.ProfileToProto(updated)}, nil
}

func (h *ProfileHandler) DeleteProfile(ctx context.Context, req *identityv1.DeleteProfileRequest) (*identityv1.DeleteProfileResponse, error) {
	if err := dependency.ValidateProfileID(req.GetId()); err != nil {
		return nil, err
	}

	if err := h.profiles.Delete(ctx, req.GetId()); err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &identityv1.DeleteProfileResponse{}, nil
}
