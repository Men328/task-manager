package service

import (
	"context"
	"errors"
	"testing"

	"taskmanager/service/identity/internal/model"
)

func TestProfileServiceCreateSeedsDefaultProject(t *testing.T) {
	projectSeeder := &stubProjectSeeder{}
	svc := NewProfileService(newStubProfiles(), &stubSeeder{}, projectSeeder)

	created, err := svc.Create(context.Background(), model.Profile{Email: "User@Example.com", IsActive: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(projectSeeder.profiles) != 1 || projectSeeder.profiles[0] != created.ID {
		t.Fatalf("phải seed project cho profile mới, nhận %v", projectSeeder.profiles)
	}
}

func TestProfileServiceCreateIgnoresProjectSeedFailure(t *testing.T) {
	projectSeeder := &stubProjectSeeder{err: errors.New("project service offline")}
	svc := NewProfileService(newStubProfiles(), &stubSeeder{}, projectSeeder)

	if _, err := svc.Create(context.Background(), model.Profile{Email: "a@example.com", IsActive: true}); err != nil {
		t.Fatalf("seed project lỗi không được làm hỏng create: %v", err)
	}
}

func TestBatchGetRejectsMoreThan500Ids(t *testing.T) {
	svc := NewProfileService(newStubProfiles(), &stubSeeder{}, &stubProjectSeeder{})

	_, err := svc.BatchGet(context.Background(), make([]string, 501))
	if !errors.Is(err, model.ErrInvalid) {
		t.Fatalf("quá 500 id phải ra ErrInvalid, nhận %v", err)
	}
}

func TestFindByEmailIgnoresCaseAndSpace(t *testing.T) {
	svc := NewProfileService(newStubProfiles(), &stubSeeder{}, &stubProjectSeeder{})
	created, err := svc.Create(context.Background(), model.Profile{Email: "an@x.io", IsActive: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	found, err := svc.FindByEmail(context.Background(), "  AN@x.io ")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found.ID != created.ID {
		t.Fatalf("phải tìm thấy %s, nhận %s", created.ID, found.ID)
	}
}
