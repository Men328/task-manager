package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"taskmanager/service/identity/internal/model"
)

func TestListByIDsSkipsDeletedAndUnknown(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryProfileRepository()
	kept, err := repo.Create(ctx, model.Profile{Email: "a@x.io", DisplayName: "A"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	removed, err := repo.Create(ctx, model.Profile{Email: "b@x.io", DisplayName: "B"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Delete(ctx, removed.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	got, err := repo.ListByIDs(ctx, []string{kept.ID, removed.ID, uuid.NewString()})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].ID != kept.ID {
		t.Fatalf("chỉ được trả profile còn tồn tại, nhận %+v", got)
	}
}
