package service

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"taskmanager/service/identity/internal/model"
)

type stubProfiles struct {
	items map[string]model.Profile
	seq   int
}

func newStubProfiles() *stubProfiles {
	return &stubProfiles{items: map[string]model.Profile{}}
}

func (s *stubProfiles) Create(_ context.Context, p model.Profile) (model.Profile, error) {
	for _, existing := range s.items {
		if strings.EqualFold(existing.Email, p.Email) {
			return model.Profile{}, model.ErrEmailExists
		}
	}
	s.seq++
	now := time.Now().UTC()
	p.ID = "profile-" + strconv.Itoa(s.seq)
	p.CreatedAt = now
	p.UpdatedAt = now
	s.items[p.ID] = p
	return p, nil
}

func (s *stubProfiles) Get(_ context.Context, id string) (model.Profile, error) {
	p, ok := s.items[id]
	if !ok {
		return model.Profile{}, model.ErrNotFound
	}
	return p, nil
}

func (s *stubProfiles) GetByEmail(_ context.Context, email string) (model.Profile, error) {
	for _, p := range s.items {
		if strings.EqualFold(p.Email, email) {
			return p, nil
		}
	}
	return model.Profile{}, model.ErrNotFound
}

func (s *stubProfiles) List(_ context.Context, limit int) ([]model.Profile, error) {
	out := make([]model.Profile, 0, len(s.items))
	for _, p := range s.items {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *stubProfiles) Update(_ context.Context, id string, upd model.ProfileUpdate) (model.Profile, error) {
	p, ok := s.items[id]
	if !ok {
		return model.Profile{}, model.ErrNotFound
	}
	if upd.DisplayName != nil {
		p.DisplayName = *upd.DisplayName
	}
	if upd.IsActive != nil {
		p.IsActive = *upd.IsActive
	}
	s.items[id] = p
	return p, nil
}

func (s *stubProfiles) TouchLastLogin(_ context.Context, id string) (model.Profile, error) {
	p, ok := s.items[id]
	if !ok {
		return model.Profile{}, model.ErrNotFound
	}
	now := time.Now().UTC()
	p.LastLoginAt = &now
	s.items[id] = p
	return p, nil
}

func (s *stubProfiles) Delete(_ context.Context, id string) error {
	if _, ok := s.items[id]; !ok {
		return model.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

type stubAuthProviders struct {
	items map[string]model.AuthProvider
	seq   int
}

func newStubAuthProviders() *stubAuthProviders {
	return &stubAuthProviders{items: map[string]model.AuthProvider{}}
}

func (s *stubAuthProviders) Create(_ context.Context, ap model.AuthProvider) (model.AuthProvider, error) {
	for _, existing := range s.items {
		if existing.Provider == ap.Provider && existing.ProviderUserID == ap.ProviderUserID {
			return existing, nil
		}
	}
	s.seq++
	ap.ID = "link-" + strconv.Itoa(s.seq)
	ap.CreatedAt = time.Now().UTC()
	s.items[ap.ID] = ap
	return ap, nil
}

func (s *stubAuthProviders) GetByProviderUserID(_ context.Context, provider string, providerUserID string) (model.AuthProvider, error) {
	for _, ap := range s.items {
		if ap.Provider == provider && ap.ProviderUserID == providerUserID {
			return ap, nil
		}
	}
	return model.AuthProvider{}, model.ErrAuthProviderNotFound
}

func (s *stubAuthProviders) ListByProfileID(_ context.Context, profileID string) ([]model.AuthProvider, error) {
	out := make([]model.AuthProvider, 0, len(s.items))
	for _, ap := range s.items {
		if ap.ProfileID == profileID {
			out = append(out, ap)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func newTestAuthService() (AuthService, *stubProfiles, *stubAuthProviders) {
	profiles := newStubProfiles()
	providers := newStubAuthProviders()
	return NewAuthService(profiles, providers), profiles, providers
}

func googleLogin(sub string, email string) model.ProviderLogin {
	return model.ProviderLogin{
		Provider:       model.ProviderGoogle,
		ProviderUserID: sub,
		Email:          email,
		DisplayName:    "Người Dùng Google",
		AvatarURL:      "https://example.com/avatar.png",
		Timezone:       model.DefaultTimezone,
		Locale:         model.DefaultLocale,
	}
}

func TestLoginWithProviderCreatesProfileOnFirstLogin(t *testing.T) {
	svc, profiles, providers := newTestAuthService()
	ctx := context.Background()

	profile, created, err := svc.LoginWithProvider(ctx, googleLogin("sub-1", "New.User@Example.com"))
	if err != nil {
		t.Fatalf("login lần đầu: %v", err)
	}
	if !created {
		t.Fatal("login lần đầu phải báo created=true")
	}
	if profile.Email != "new.user@example.com" {
		t.Fatalf("email phải được chuẩn hoá chữ thường, nhận %q", profile.Email)
	}
	if profile.LastLoginAt == nil {
		t.Fatal("login phải set last_login_at")
	}
	if len(profiles.items) != 1 {
		t.Fatalf("phải có đúng 1 profile, nhận %d", len(profiles.items))
	}
	if len(providers.items) != 1 {
		t.Fatalf("phải có đúng 1 liên kết provider, nhận %d", len(providers.items))
	}
}

func TestLoginWithProviderReusesExistingLink(t *testing.T) {
	svc, profiles, _ := newTestAuthService()
	ctx := context.Background()

	first, _, err := svc.LoginWithProvider(ctx, googleLogin("sub-1", "user@example.com"))
	if err != nil {
		t.Fatalf("login lần đầu: %v", err)
	}

	second, created, err := svc.LoginWithProvider(ctx, googleLogin("sub-1", "user@example.com"))
	if err != nil {
		t.Fatalf("login lần hai: %v", err)
	}
	if created {
		t.Fatal("login lần hai không được báo created")
	}
	if second.ID != first.ID {
		t.Fatalf("login lần hai phải trả cùng profile: %s != %s", second.ID, first.ID)
	}
	if len(profiles.items) != 1 {
		t.Fatalf("không được tạo profile mới, nhận %d", len(profiles.items))
	}
}

func TestLoginWithProviderLinksExistingEmail(t *testing.T) {
	svc, profiles, providers := newTestAuthService()
	ctx := context.Background()

	existing, err := profiles.Create(ctx, model.Profile{
		Email:       "user@example.com",
		DisplayName: "Đã Có Sẵn",
		IsActive:    true,
	})
	if err != nil {
		t.Fatalf("tạo profile trước: %v", err)
	}

	profile, created, err := svc.LoginWithProvider(ctx, googleLogin("sub-google", "USER@example.com"))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if created {
		t.Fatal("email đã tồn tại thì không được báo created")
	}
	if profile.ID != existing.ID {
		t.Fatalf("phải liên kết vào profile sẵn có: %s != %s", profile.ID, existing.ID)
	}
	if profile.DisplayName != "Đã Có Sẵn" {
		t.Fatalf("không được ghi đè display_name sẵn có, nhận %q", profile.DisplayName)
	}
	if len(providers.items) != 1 {
		t.Fatalf("phải tạo 1 liên kết provider, nhận %d", len(providers.items))
	}
}

func TestLoginWithProviderRejectsInactiveProfile(t *testing.T) {
	svc, profiles, _ := newTestAuthService()
	ctx := context.Background()

	inactive, err := profiles.Create(ctx, model.Profile{
		Email:       "khoa@example.com",
		DisplayName: "Bị Khoá",
		IsActive:    false,
	})
	if err != nil {
		t.Fatalf("tạo profile: %v", err)
	}

	_, _, err = svc.LoginWithProvider(ctx, googleLogin("sub-khoa", inactive.Email))
	if err != model.ErrProfileInactive {
		t.Fatalf("profile inactive phải trả ErrProfileInactive, nhận %v", err)
	}
}

func TestListProvidersRequiresExistingProfile(t *testing.T) {
	svc, profiles, _ := newTestAuthService()
	ctx := context.Background()

	if _, err := svc.ListProviders(ctx, "khong-ton-tai"); err != model.ErrNotFound {
		t.Fatalf("profile không tồn tại phải trả ErrNotFound, nhận %v", err)
	}

	profile, err := profiles.Create(ctx, model.Profile{
		Email:       "user@example.com",
		DisplayName: "Người Dùng",
		IsActive:    true,
	})
	if err != nil {
		t.Fatalf("tạo profile: %v", err)
	}
	if _, _, err := svc.LoginWithProvider(ctx, googleLogin("sub-1", profile.Email)); err != nil {
		t.Fatalf("login: %v", err)
	}

	links, err := svc.ListProviders(ctx, profile.ID)
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if len(links) != 1 || links[0].Provider != model.ProviderGoogle {
		t.Fatalf("phải trả 1 liên kết google, nhận %+v", links)
	}
}
