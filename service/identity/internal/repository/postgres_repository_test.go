package repository

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/identity/internal/model"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL chưa được đặt, bỏ qua test postgres")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("khởi tạo pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping postgres: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func uniqueEmail(t *testing.T) string {
	t.Helper()
	return "test-" + time.Now().UTC().Format("20060102150405.000000000") + "@example.com"
}

func cleanupProfile(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()
	if id == "" {
		return
	}
	ctx := context.Background()
	_, _ = pool.Exec(ctx, `DELETE FROM identity.auth_providers WHERE profile_id = $1::uuid`, id)
	_, _ = pool.Exec(ctx, `DELETE FROM identity.profiles WHERE id = $1::uuid`, id)
}

func TestPostgresProfileRepositoryLifecycle(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresProfileRepository(pool)
	ctx := context.Background()
	email := uniqueEmail(t)

	created, err := repo.Create(ctx, model.Profile{
		Email:       email,
		DisplayName: "Người Dùng Test",
		Timezone:    model.DefaultTimezone,
		Locale:      model.DefaultLocale,
		IsActive:    true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() { cleanupProfile(t, pool, created.ID) })

	if created.ID == "" {
		t.Fatal("create không trả về id")
	}
	if created.LastLoginAt != nil {
		t.Fatalf("last_login_at phải NULL khi mới tạo, nhận %v", created.LastLoginAt)
	}

	if _, err := repo.Create(ctx, model.Profile{
		Email:       email,
		DisplayName: "Trùng Email",
		Timezone:    model.DefaultTimezone,
		Locale:      model.DefaultLocale,
		IsActive:    true,
	}); !errors.Is(err, model.ErrEmailExists) {
		t.Fatalf("email trùng phải trả ErrEmailExists, nhận %v", err)
	}

	fetched, err := repo.GetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if fetched.ID != created.ID {
		t.Fatalf("get by email trả sai profile: %s != %s", fetched.ID, created.ID)
	}

	upper, err := repo.GetByEmail(ctx, strings.ToUpper(email))
	if err != nil || upper.ID != created.ID {
		t.Fatalf("get by email phải không phân biệt hoa thường, nhận %v / %v", upper.ID, err)
	}

	touched, err := repo.TouchLastLogin(ctx, created.ID)
	if err != nil {
		t.Fatalf("touch last login: %v", err)
	}
	if touched.LastLoginAt == nil {
		t.Fatal("touch last login phải set last_login_at")
	}

	name := "Tên Mới"
	updated, err := repo.Update(ctx, created.ID, model.ProfileUpdate{DisplayName: &name})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.DisplayName != name {
		t.Fatalf("update không đổi display_name: %s", updated.DisplayName)
	}
	if updated.AvatarURL != created.AvatarURL {
		t.Fatalf("update không được ghi đè field không truyền: %q", updated.AvatarURL)
	}

	list, err := repo.List(ctx, 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !containsProfile(list, created.ID) {
		t.Fatal("list không chứa profile vừa tạo")
	}

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get(ctx, created.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("profile đã xoá mềm phải là ErrNotFound, nhận %v", err)
	}
	if _, err := repo.GetByEmail(ctx, email); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("email của profile đã xoá phải là ErrNotFound, nhận %v", err)
	}
	if err := repo.Delete(ctx, created.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("xoá lần hai phải là ErrNotFound, nhận %v", err)
	}
}

func TestPostgresProfileRepositoryInvalidID(t *testing.T) {
	pool := testPool(t)
	repo := NewPostgresProfileRepository(pool)

	if _, err := repo.Get(context.Background(), "khong-phai-uuid"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("id không hợp lệ phải quy về ErrNotFound, nhận %v", err)
	}
}

func TestPostgresAuthProviderRepository(t *testing.T) {
	pool := testPool(t)
	profiles := NewPostgresProfileRepository(pool)
	providers := NewPostgresAuthProviderRepository(pool)
	ctx := context.Background()

	profile, err := profiles.Create(ctx, model.Profile{
		Email:       uniqueEmail(t),
		DisplayName: "Chủ Tài Khoản",
		Timezone:    model.DefaultTimezone,
		Locale:      model.DefaultLocale,
		IsActive:    true,
	})
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	t.Cleanup(func() { cleanupProfile(t, pool, profile.ID) })

	providerUserID := "google-sub-" + time.Now().UTC().Format("150405.000000000")

	link, err := providers.Create(ctx, model.AuthProvider{
		ProfileID:      profile.ID,
		Provider:       model.ProviderGoogle,
		ProviderUserID: providerUserID,
	})
	if err != nil {
		t.Fatalf("create auth provider: %v", err)
	}
	if link.ProfileID != profile.ID {
		t.Fatalf("auth provider trỏ sai profile: %s", link.ProfileID)
	}

	fetched, err := providers.GetByProviderUserID(ctx, model.ProviderGoogle, providerUserID)
	if err != nil {
		t.Fatalf("get by provider user id: %v", err)
	}
	if fetched.ID != link.ID {
		t.Fatalf("get trả sai bản ghi: %s != %s", fetched.ID, link.ID)
	}

	again, err := providers.Create(ctx, model.AuthProvider{
		ProfileID:      profile.ID,
		Provider:       model.ProviderGoogle,
		ProviderUserID: providerUserID,
	})
	if err != nil {
		t.Fatalf("create lần hai phải idempotent: %v", err)
	}
	if again.ID != link.ID {
		t.Fatalf("create lần hai phải trả cùng bản ghi, nhận %s", again.ID)
	}

	links, err := providers.ListByProfileID(ctx, profile.ID)
	if err != nil {
		t.Fatalf("list by profile: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("list phải trả đúng 1 liên kết, nhận %d", len(links))
	}

	if _, err := providers.GetByProviderUserID(ctx, model.ProviderGoogle, "khong-ton-tai"); !errors.Is(err, model.ErrAuthProviderNotFound) {
		t.Fatalf("liên kết không tồn tại phải là ErrAuthProviderNotFound, nhận %v", err)
	}
}

func containsProfile(items []model.Profile, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}
