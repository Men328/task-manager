package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/social/internal/model"
)

func TestMapFriendshipError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want error
	}{
		{"trùng cặp", &pgconn.PgError{Code: "23505", ConstraintName: "uq_friendships_pair"}, model.ErrDuplicate},
		{"profile không tồn tại", &pgconn.PgError{Code: "23503"}, model.ErrNotFound},
		{"id sai định dạng", &pgconn.PgError{Code: "22P02"}, model.ErrNotFound},
		{"không có dòng", pgx.ErrNoRows, model.ErrNotFound},
	}
	for _, tc := range cases {
		if got := mapFriendshipError(tc.err); !errors.Is(got, tc.want) {
			t.Fatalf("%s: phải ra %v, nhận %v", tc.name, tc.want, got)
		}
	}
}

type contractRepo interface {
	Create(ctx context.Context, requesterID, addresseeID string) (model.Friendship, error)
	Get(ctx context.Context, id string) (model.Friendship, error)
	GetByPair(ctx context.Context, a, b string) (model.Friendship, error)
	Accept(ctx context.Context, id, addresseeID string) (model.Friendship, error)
	Delete(ctx context.Context, id string, status model.Status) error
	ListAccepted(ctx context.Context, profileID string) ([]model.Friendship, error)
	ListPending(ctx context.Context, profileID string) ([]model.Friendship, error)
}

func forEachRepo(t *testing.T, run func(t *testing.T, repo contractRepo, a, b string)) {
	t.Helper()
	t.Run("in-memory", func(t *testing.T) {
		run(t, NewInMemoryFriendshipRepository(), uuid.NewString(), uuid.NewString())
	})
	t.Run("postgres", func(t *testing.T) {
		repo, a, b := postgresRepo(t)
		run(t, repo, a, b)
	})
}

func postgresRepo(t *testing.T) (*PostgresFriendshipRepository, string, string) {
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
	t.Cleanup(pool.Close)

	a, b := uuid.NewString(), uuid.NewString()
	for _, id := range []string{a, b} {
		if _, err := pool.Exec(ctx, `INSERT INTO identity.profiles (id, email, display_name) VALUES ($1, $2, 'T')`, id, id+"@x.io"); err != nil {
			t.Fatalf("tạo profile: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM identity.profiles WHERE id = ANY($1::uuid[])`, []string{a, b})
	})
	return NewPostgresFriendshipRepository(pool), a, b
}

func TestRepoPairIsUniqueInBothDirections(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo contractRepo, a, b string) {
		ctx := context.Background()

		if _, err := repo.Create(ctx, a, b); err != nil {
			t.Fatalf("tạo: %v", err)
		}
		if _, err := repo.Create(ctx, b, a); !errors.Is(err, model.ErrDuplicate) {
			t.Fatalf("chiều ngược phải ra ErrDuplicate, nhận %v", err)
		}
		row, err := repo.GetByPair(ctx, b, a)
		if err != nil || row.RequesterID != a {
			t.Fatalf("GetByPair phải tìm được ở cả 2 chiều, nhận %+v, %v", row, err)
		}
	})
}

func TestRepoAcceptOnlyPendingAddressee(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo contractRepo, a, b string) {
		ctx := context.Background()
		row, err := repo.Create(ctx, a, b)
		if err != nil {
			t.Fatalf("tạo: %v", err)
		}

		if _, err := repo.Accept(ctx, row.ID, a); !errors.Is(err, model.ErrNotFound) {
			t.Fatalf("người gửi không được chấp nhận, nhận %v", err)
		}
		accepted, err := repo.Accept(ctx, row.ID, b)
		if err != nil || accepted.Status != model.StatusAccepted || accepted.AcceptedAt == nil {
			t.Fatalf("chấp nhận: %+v, %v", accepted, err)
		}
		if _, err := repo.Accept(ctx, row.ID, b); !errors.Is(err, model.ErrNotFound) {
			t.Fatalf("chấp nhận lần 2 phải ra ErrNotFound, nhận %v", err)
		}
	})
}

func TestRepoDeleteRequiresStatus(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo contractRepo, a, b string) {
		ctx := context.Background()
		row, err := repo.Create(ctx, a, b)
		if err != nil {
			t.Fatalf("tạo: %v", err)
		}

		if err := repo.Delete(ctx, row.ID, model.StatusAccepted); !errors.Is(err, model.ErrNotFound) {
			t.Fatalf("sai status phải ra ErrNotFound, nhận %v", err)
		}
		if err := repo.Delete(ctx, row.ID, model.StatusPending); err != nil {
			t.Fatalf("xoá: %v", err)
		}
		if _, err := repo.Get(ctx, row.ID); !errors.Is(err, model.ErrNotFound) {
			t.Fatalf("dòng phải bị xoá, nhận %v", err)
		}
	})
}

func TestRepoListsByStatus(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo contractRepo, a, b string) {
		ctx := context.Background()
		row, err := repo.Create(ctx, a, b)
		if err != nil {
			t.Fatalf("tạo: %v", err)
		}

		pending, err := repo.ListPending(ctx, b)
		if err != nil || len(pending) != 1 || pending[0].ID != row.ID {
			t.Fatalf("pending: %+v, %v", pending, err)
		}
		if _, err := repo.Accept(ctx, row.ID, b); err != nil {
			t.Fatalf("chấp nhận: %v", err)
		}
		accepted, err := repo.ListAccepted(ctx, a)
		if err != nil || len(accepted) != 1 || accepted[0].AcceptedAt == nil {
			t.Fatalf("accepted: %+v, %v", accepted, err)
		}
	})
}

func TestRepoMalformedIDIsNotFound(t *testing.T) {
	forEachRepo(t, func(t *testing.T, repo contractRepo, a, b string) {
		ctx := context.Background()

		if _, err := repo.Get(ctx, "not-a-uuid"); !errors.Is(err, model.ErrNotFound) {
			t.Fatalf("Get phải ra ErrNotFound, nhận %v", err)
		}
		if _, err := repo.Accept(ctx, "not-a-uuid", b); !errors.Is(err, model.ErrNotFound) {
			t.Fatalf("Accept phải ra ErrNotFound, nhận %v", err)
		}
		if _, err := repo.GetByPair(ctx, b, "not-a-uuid"); !errors.Is(err, model.ErrNotFound) {
			t.Fatalf("GetByPair phải ra ErrNotFound, nhận %v", err)
		}
	})
}
