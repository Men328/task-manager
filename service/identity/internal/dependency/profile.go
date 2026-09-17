package dependency

import (
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	identityv1 "taskmanager/common/gen/go/identity/v1"
	"taskmanager/service/identity/internal/model"
)

const (
	defaultTimezone = "Asia/Ho_Chi_Minh"
	defaultLocale   = "vi"
)

func ValidateCreateProfile(req *identityv1.CreateProfileRequest) error {
	if strings.TrimSpace(req.GetEmail()) == "" {
		return errorcode.Error(errorcode.IdentityEmailRequired, "email là bắt buộc")
	}
	if strings.TrimSpace(req.GetDisplayName()) == "" {
		return errorcode.Error(errorcode.IdentityDisplayNameRequired, "display_name là bắt buộc")
	}
	return nil
}

func ValidateProfileID(id string) error {
	if id == "" {
		return errorcode.Error(errorcode.IdentityProfileIDRequired, "id là bắt buộc")
	}
	return nil
}

func ProfileFromCreateRequest(req *identityv1.CreateProfileRequest) model.Profile {
	return model.Profile{
		Email:       req.GetEmail(),
		DisplayName: req.GetDisplayName(),
		AvatarURL:   req.GetAvatarUrl(),
		Timezone:    orDefault(req.GetTimezone(), defaultTimezone),
		Locale:      orDefault(req.GetLocale(), defaultLocale),
		IsActive:    true,
	}
}

func ProfileUpdateFromRequest(req *identityv1.UpdateProfileRequest) model.ProfileUpdate {
	return model.ProfileUpdate{
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarUrl,
		Timezone:    req.Timezone,
		Locale:      req.Locale,
		IsActive:    req.IsActive,
	}
}

func ProfileToProto(p model.Profile) *identityv1.Profile {
	return &identityv1.Profile{
		Id:          p.ID,
		Email:       p.Email,
		DisplayName: p.DisplayName,
		AvatarUrl:   p.AvatarURL,
		Timezone:    p.Timezone,
		Locale:      p.Locale,
		IsActive:    p.IsActive,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
}

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
