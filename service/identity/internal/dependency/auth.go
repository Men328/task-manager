package dependency

import (
	"strings"

	"taskmanager/common/errorcode"
	identityv1 "taskmanager/common/gen/go/identity/v1"
	"taskmanager/service/identity/internal/model"
)

func ValidateLoginWithProvider(req *identityv1.LoginWithProviderRequest) error {
	provider := strings.ToLower(strings.TrimSpace(req.GetProvider()))
	if provider == "" {
		return errorcode.Error(errorcode.IdentityAuthProviderRequired, "provider là bắt buộc")
	}
	if provider != model.ProviderGoogle {
		return errorcode.Error(errorcode.IdentityAuthProviderUnsupported, "provider không được hỗ trợ: "+provider)
	}
	if strings.TrimSpace(req.GetProviderUserId()) == "" {
		return errorcode.Error(errorcode.IdentityAuthProviderUserIDRequired, "provider_user_id là bắt buộc")
	}
	if strings.TrimSpace(req.GetEmail()) == "" {
		return errorcode.Error(errorcode.IdentityEmailRequired, "email là bắt buộc")
	}
	return nil
}

func ProviderLoginFromRequest(req *identityv1.LoginWithProviderRequest) model.ProviderLogin {
	email := strings.ToLower(strings.TrimSpace(req.GetEmail()))

	return model.ProviderLogin{
		Provider:       strings.ToLower(strings.TrimSpace(req.GetProvider())),
		ProviderUserID: strings.TrimSpace(req.GetProviderUserId()),
		Email:          email,
		DisplayName:    orDefault(strings.TrimSpace(req.GetDisplayName()), emailLocalPart(email)),
		AvatarURL:      strings.TrimSpace(req.GetAvatarUrl()),
		Timezone:       orDefault(req.GetTimezone(), model.DefaultTimezone),
		Locale:         orDefault(req.GetLocale(), model.DefaultLocale),
	}
}

func emailLocalPart(email string) string {
	local, _, found := strings.Cut(email, "@")
	if !found || strings.TrimSpace(local) == "" {
		return email
	}
	return local
}
