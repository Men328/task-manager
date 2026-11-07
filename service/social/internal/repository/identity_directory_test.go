package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "taskmanager/common/gen/go/identity/v1"
	"taskmanager/service/social/internal/model"
)

type stubProfileClient struct {
	identityv1.ProfileServiceClient
	err        error
	batchSizes []int
}

func (c *stubProfileClient) FindProfileByEmail(_ context.Context, in *identityv1.FindProfileByEmailRequest, _ ...grpc.CallOption) (*identityv1.FindProfileByEmailResponse, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &identityv1.FindProfileByEmailResponse{Profile: &identityv1.Profile{
		Id: "p1", Email: in.GetEmail(), DisplayName: "An", AvatarUrl: "a.png", Timezone: "UTC",
	}}, nil
}

func (c *stubProfileClient) BatchGetProfiles(_ context.Context, in *identityv1.BatchGetProfilesRequest, _ ...grpc.CallOption) (*identityv1.BatchGetProfilesResponse, error) {
	if c.err != nil {
		return nil, c.err
	}
	c.batchSizes = append(c.batchSizes, len(in.GetIds()))
	out := make([]*identityv1.Profile, 0, len(in.GetIds()))
	for _, id := range in.GetIds() {
		out = append(out, &identityv1.Profile{Id: id})
	}
	return &identityv1.BatchGetProfilesResponse{Profiles: out}, nil
}

func TestIdentityDirectoryMapsProfile(t *testing.T) {
	directory := NewIdentityDirectory(&stubProfileClient{}, time.Second)

	got, err := directory.FindByEmail(context.Background(), "an@x.io")
	if err != nil {
		t.Fatalf("tìm: %v", err)
	}
	want := model.Profile{ID: "p1", Email: "an@x.io", DisplayName: "An", AvatarURL: "a.png"}
	if got != want {
		t.Fatalf("phải ra %+v, nhận %+v", want, got)
	}
}

func TestIdentityDirectoryMapsStatus(t *testing.T) {
	cases := []struct {
		code codes.Code
		want error
	}{
		{codes.NotFound, model.ErrNotFound},
		{codes.Unavailable, model.ErrUnavailable},
		{codes.DeadlineExceeded, model.ErrUnavailable},
	}
	for _, tc := range cases {
		directory := NewIdentityDirectory(&stubProfileClient{err: status.Error(tc.code, "x")}, time.Second)
		if _, err := directory.FindByEmail(context.Background(), "an@x.io"); !errors.Is(err, tc.want) {
			t.Fatalf("%v: FindByEmail phải ra %v, nhận %v", tc.code, tc.want, err)
		}
	}

	directory := NewIdentityDirectory(&stubProfileClient{err: status.Error(codes.Unavailable, "x")}, time.Second)
	if _, err := directory.GetByIDs(context.Background(), []string{"a"}); !errors.Is(err, model.ErrUnavailable) {
		t.Fatalf("GetByIDs phải ra ErrUnavailable, nhận %v", err)
	}
}

func TestIdentityDirectoryChunksLargeLookups(t *testing.T) {
	client := &stubProfileClient{}
	directory := NewIdentityDirectory(client, time.Second)
	ids := make([]string, 501)
	for i := range ids {
		ids[i] = fmt.Sprintf("id-%d", i)
	}

	got, err := directory.GetByIDs(context.Background(), ids)
	if err != nil {
		t.Fatalf("tra: %v", err)
	}
	if len(got) != 501 {
		t.Fatalf("phải đủ 501 profile, nhận %d", len(got))
	}
	if len(client.batchSizes) != 2 || client.batchSizes[0] != 500 || client.batchSizes[1] != 1 {
		t.Fatalf("phải chia lô 500 + 1, nhận %v", client.batchSizes)
	}
}
