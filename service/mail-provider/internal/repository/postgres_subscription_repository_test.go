package repository

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/mail-provider/internal/model"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
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

func createTestProfile(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	ctx := context.Background()
	email := "mail-test-" + time.Now().UTC().Format("20060102150405.000000000") + "@example.com"

	var id string
	err := pool.QueryRow(ctx,
		`INSERT INTO identity.profiles (email, display_name) VALUES ($1, $2) RETURNING id::text`,
		email, "Mail Provider Test",
	).Scan(&id)
	if err != nil {
		t.Fatalf("tạo profile test: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM identity.profiles WHERE id = $1::uuid`, id)
	})
	return id
}

func TestPostgresSubscriptionLifecycle(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPostgresSubscriptionRepository(pool)
	ctx := context.Background()
	profileID := createTestProfile(t, pool)

	expiresAt := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)

	created, err := repo.Upsert(ctx, model.Subscription{
		ProfileID:            profileID,
		Email:                "User@Example.com",
		RefreshToken:         "refresh-1",
		AccessToken:          "access-1",
		AccessTokenExpiresAt: expiresAt,
		HistoryID:            "100",
		WatchExpiresAt:       expiresAt,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if created.Email != "user@example.com" {
		t.Fatalf("email phải được chuẩn hoá chữ thường, nhận %q", created.Email)
	}
	if created.RefreshToken != "refresh-1" || created.AccessToken != "access-1" {
		t.Fatalf("upsert không lưu token: %+v", created)
	}
	if created.HistoryID != "100" {
		t.Fatalf("upsert không lưu history_id, nhận %q", created.HistoryID)
	}
	if !created.AccessTokenExpiresAt.Equal(expiresAt) || !created.WatchExpiresAt.Equal(expiresAt) {
		t.Fatalf("upsert lưu sai hạn: %+v", created)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatalf("upsert phải set created_at/updated_at: %+v", created)
	}

	byID, err := repo.GetByProfileID(ctx, profileID)
	if err != nil || byID.HistoryID != "100" {
		t.Fatalf("get by profile: %+v / %v", byID, err)
	}

	byEmail, err := repo.GetByEmail(ctx, "USER@example.com")
	if err != nil || byEmail.ProfileID != profileID {
		t.Fatalf("get by email phải không phân biệt hoa thường: %+v / %v", byEmail, err)
	}

	if _, err := repo.GetByProfileID(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("profile lạ phải là ErrNotFound, nhận %v", err)
	}
	if _, err := repo.GetByProfileID(ctx, "khong-phai-uuid"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("profile id không hợp lệ phải là ErrNotFound, nhận %v", err)
	}
	if _, err := repo.GetByEmail(ctx, "missing@example.com"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("email lạ phải là ErrNotFound, nhận %v", err)
	}

	watchExpiresAt := time.Now().UTC().Add(6 * 24 * time.Hour).Truncate(time.Microsecond)
	watched, err := repo.UpdateWatch(ctx, profileID, "500", watchExpiresAt)
	if err != nil {
		t.Fatalf("update watch: %v", err)
	}
	if watched.HistoryID != "500" || !watched.WatchExpiresAt.Equal(watchExpiresAt) {
		t.Fatalf("update watch sai: %+v", watched)
	}
	if watched.RefreshToken != "refresh-1" {
		t.Fatalf("update watch không được đụng tới refresh token: %+v", watched)
	}

	tokenExpiry := time.Now().UTC().Add(30 * time.Minute).Truncate(time.Microsecond)
	tokened, err := repo.UpdateTokens(ctx, profileID, model.Token{
		AccessToken:  "access-2",
		RefreshToken: "refresh-2",
		ExpiresAt:    tokenExpiry,
	})
	if err != nil {
		t.Fatalf("update tokens: %v", err)
	}
	if tokened.AccessToken != "access-2" || tokened.RefreshToken != "refresh-2" {
		t.Fatalf("update tokens sai: %+v", tokened)
	}
	if !tokened.AccessTokenExpiresAt.Equal(tokenExpiry) {
		t.Fatalf("update tokens sai hạn: %+v", tokened)
	}
	if tokened.HistoryID != "500" {
		t.Fatalf("update tokens không được đụng tới checkpoint: %+v", tokened)
	}

	items, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !containsSubscription(items, profileID) {
		t.Fatal("list không chứa subscription vừa tạo")
	}

	if err := repo.Delete(ctx, profileID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.GetByProfileID(ctx, profileID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("subscription đã ngắt phải là ErrNotFound, nhận %v", err)
	}
	if _, err := repo.GetByEmail(ctx, "user@example.com"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("email của subscription đã ngắt phải là ErrNotFound, nhận %v", err)
	}
	items, err = repo.List(ctx)
	if err != nil {
		t.Fatalf("list sau delete: %v", err)
	}
	if containsSubscription(items, profileID) {
		t.Fatal("list không được chứa subscription đã ngắt")
	}
	if err := repo.Delete(ctx, profileID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("delete lần hai phải là ErrNotFound, nhận %v", err)
	}
}

func TestPostgresSubscriptionUpsertKeepsCreatedAtAndTokens(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPostgresSubscriptionRepository(pool)
	ctx := context.Background()
	profileID := createTestProfile(t, pool)

	first, err := repo.Upsert(ctx, model.Subscription{
		ProfileID:    profileID,
		Email:        "keep@example.com",
		RefreshToken: "refresh-keep",
		AccessToken:  "access-keep",
		HistoryID:    "10",
	})
	if err != nil {
		t.Fatalf("upsert lần 1: %v", err)
	}

	time.Sleep(2 * time.Millisecond)

	second, err := repo.Upsert(ctx, model.Subscription{
		ProfileID: profileID,
		Email:     "keep@example.com",
		HistoryID: "20",
	})
	if err != nil {
		t.Fatalf("upsert lần 2: %v", err)
	}
	if !second.CreatedAt.Equal(first.CreatedAt) {
		t.Fatalf("upsert không được đổi created_at: %s != %s", second.CreatedAt, first.CreatedAt)
	}
	if second.RefreshToken != "refresh-keep" || second.AccessToken != "access-keep" {
		t.Fatalf("upsert không được xoá token cũ khi field để trống: %+v", second)
	}
	if second.HistoryID != "20" {
		t.Fatalf("upsert phải cập nhật checkpoint mới, nhận %q", second.HistoryID)
	}
	if !second.UpdatedAt.After(first.UpdatedAt) {
		t.Fatalf("upsert phải bump updated_at: %s", second.UpdatedAt)
	}
}

func TestPostgresSubscriptionReconnectAfterDelete(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPostgresSubscriptionRepository(pool)
	ctx := context.Background()
	profileID := createTestProfile(t, pool)

	if _, err := repo.Upsert(ctx, model.Subscription{ProfileID: profileID, Email: "old@example.com"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := repo.Delete(ctx, profileID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	reconnected, err := repo.Upsert(ctx, model.Subscription{
		ProfileID:    profileID,
		Email:        "new@example.com",
		RefreshToken: "refresh-new",
		HistoryID:    "77",
	})
	if err != nil {
		t.Fatalf("upsert sau khi ngắt: %v", err)
	}
	if reconnected.Email != "new@example.com" || reconnected.RefreshToken != "refresh-new" {
		t.Fatalf("kết nối lại phải ghi đè email/token: %+v", reconnected)
	}
	if _, err := repo.GetByEmail(ctx, "old@example.com"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("email cũ phải bị xoá khỏi index, nhận %v", err)
	}
	if found, err := repo.GetByEmail(ctx, strings.ToUpper("new@example.com")); err != nil || found.ProfileID != profileID {
		t.Fatalf("email mới phải tra được: %+v / %v", found, err)
	}
}

func TestPostgresSubscriptionWithoutHistoryID(t *testing.T) {
	pool := newTestPool(t)
	repo := NewPostgresSubscriptionRepository(pool)
	ctx := context.Background()
	profileID := createTestProfile(t, pool)

	created, err := repo.Upsert(ctx, model.Subscription{
		ProfileID:    profileID,
		Email:        "nohist@example.com",
		RefreshToken: "refresh-nohist",
	})
	if err != nil {
		t.Fatalf("upsert không có history_id: %v", err)
	}
	if created.HistoryID != "" || !created.WatchExpiresAt.IsZero() {
		t.Fatalf("chưa có checkpoint thì phải để trống: %+v", created)
	}

	watchExpiresAt := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	watched, err := repo.UpdateWatch(ctx, profileID, "", watchExpiresAt)
	if err != nil {
		t.Fatalf("update watch không có history_id: %v", err)
	}
	if watched.HistoryID != "" || !watched.WatchExpiresAt.Equal(watchExpiresAt) {
		t.Fatalf("update watch không có history_id sai: %+v", watched)
	}

	if _, err := repo.UpdateWatch(ctx, "00000000-0000-0000-0000-000000000000", "1", watchExpiresAt); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("update watch profile lạ phải là ErrNotFound, nhận %v", err)
	}
	if _, err := repo.UpdateTokens(ctx, "00000000-0000-0000-0000-000000000000", model.Token{}); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("update tokens profile lạ phải là ErrNotFound, nhận %v", err)
	}
}

func containsSubscription(items []model.Subscription, profileID string) bool {
	for _, item := range items {
		if item.ProfileID == profileID {
			return true
		}
	}
	return false
}
