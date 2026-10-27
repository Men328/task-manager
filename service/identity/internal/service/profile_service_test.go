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
