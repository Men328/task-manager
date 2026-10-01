package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

func TestInMemorySubscriptionUpsertAndLookup(t *testing.T) {
	repo := NewInMemorySubscriptionRepository()
	ctx := context.Background()

	created, err := repo.Upsert(ctx, model.Subscription{
		ProfileID: "p1",
		Email:     "User@Example.com",
		HistoryID: "10",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatal("upsert phải set created_at/updated_at")
	}

	byID, err := repo.GetByProfileID(ctx, "p1")
	if err != nil || byID.Email != "User@Example.com" {
		t.Fatalf("get by profile: %+v / %v", byID, err)
	}

	byEmail, err := repo.GetByEmail(ctx, "user@example.com")
	if err != nil || byEmail.ProfileID != "p1" {
		t.Fatalf("get by email phải không phân biệt hoa thường: %+v / %v", byEmail, err)
	}

	if _, err := repo.GetByProfileID(ctx, "missing"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("profile lạ phải trả ErrNotFound, nhận %v", err)
	}
	if _, err := repo.GetByEmail(ctx, "missing@example.com"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("email lạ phải trả ErrNotFound, nhận %v", err)
	}
}

func TestInMemorySubscriptionUpsertKeepsCreatedAt(t *testing.T) {
	repo := NewInMemorySubscriptionRepository()
	ctx := context.Background()

	first, err := repo.Upsert(ctx, model.Subscription{ProfileID: "p1", Email: "a@example.com"})
	if err != nil {
		t.Fatalf("upsert lần 1: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	second, err := repo.Upsert(ctx, model.Subscription{ProfileID: "p1", Email: "a@example.com", HistoryID: "99"})
	if err != nil {
		t.Fatalf("upsert lần 2: %v", err)
	}
	if !second.CreatedAt.Equal(first.CreatedAt) {
		t.Fatalf("upsert không được đổi created_at: %s != %s", second.CreatedAt, first.CreatedAt)
	}
	if second.HistoryID != "99" {
		t.Fatalf("upsert phải cập nhật field mới, nhận %q", second.HistoryID)
	}
}

func TestInMemorySubscriptionEmailChangeRebuildsIndex(t *testing.T) {
	repo := NewInMemorySubscriptionRepository()
	ctx := context.Background()

	if _, err := repo.Upsert(ctx, model.Subscription{ProfileID: "p1", Email: "old@example.com"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if _, err := repo.Upsert(ctx, model.Subscription{ProfileID: "p1", Email: "new@example.com"}); err != nil {
		t.Fatalf("upsert đổi email: %v", err)
	}

	if _, err := repo.GetByEmail(ctx, "old@example.com"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("email cũ phải bị xoá khỏi index, nhận %v", err)
	}
	if _, err := repo.GetByEmail(ctx, "new@example.com"); err != nil {
		t.Fatalf("email mới phải tra được: %v", err)
	}
}

func TestInMemorySubscriptionUpdateWatchAndTokens(t *testing.T) {
	repo := NewInMemorySubscriptionRepository()
	ctx := context.Background()

	if _, err := repo.Upsert(ctx, model.Subscription{ProfileID: "p1", Email: "a@example.com"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	expiresAt := time.Now().Add(time.Hour).UTC()
	updated, err := repo.UpdateWatch(ctx, "p1", "500", expiresAt)
	if err != nil {
		t.Fatalf("update watch: %v", err)
	}
	if updated.HistoryID != "500" || !updated.WatchExpiresAt.Equal(expiresAt) {
		t.Fatalf("update watch sai: %+v", updated)
	}

	tokenExpiry := time.Now().Add(30 * time.Minute).UTC()
	tokened, err := repo.UpdateTokens(ctx, "p1", model.Token{AccessToken: "new", RefreshToken: "r2", ExpiresAt: tokenExpiry})
	if err != nil {
		t.Fatalf("update tokens: %v", err)
	}
	if tokened.AccessToken != "new" || tokened.RefreshToken != "r2" || !tokened.AccessTokenExpiresAt.Equal(tokenExpiry) {
		t.Fatalf("update tokens sai: %+v", tokened)
	}

	if _, err := repo.UpdateWatch(ctx, "missing", "1", expiresAt); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("update watch profile lạ phải là ErrNotFound, nhận %v", err)
	}
	if _, err := repo.UpdateTokens(ctx, "missing", model.Token{}); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("update tokens profile lạ phải là ErrNotFound, nhận %v", err)
	}
}

func TestInMemorySubscriptionDeleteRemovesEmailIndex(t *testing.T) {
	repo := NewInMemorySubscriptionRepository()
	ctx := context.Background()

	if _, err := repo.Upsert(ctx, model.Subscription{ProfileID: "p1", Email: "a@example.com"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := repo.Delete(ctx, "p1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.GetByEmail(ctx, "a@example.com"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("email phải bị xoá khỏi index, nhận %v", err)
	}
	if err := repo.Delete(ctx, "p1"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("delete lần hai phải là ErrNotFound, nhận %v", err)
	}
}

func TestInMemorySubscriptionList(t *testing.T) {
	repo := NewInMemorySubscriptionRepository()
	ctx := context.Background()

	if _, err := repo.Upsert(ctx, model.Subscription{ProfileID: "p1", Email: "a@example.com"}); err != nil {
		t.Fatalf("upsert 1: %v", err)
	}
	if _, err := repo.Upsert(ctx, model.Subscription{ProfileID: "p2", Email: "b@example.com"}); err != nil {
		t.Fatalf("upsert 2: %v", err)
	}

	items, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("list phải trả 2 subscription, nhận %d", len(items))
	}
}
