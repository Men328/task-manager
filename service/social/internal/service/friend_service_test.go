package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"taskmanager/service/social/internal/model"
	"taskmanager/service/social/internal/service"
)

const (
	alice = "a0000000-0000-0000-0000-000000000001"
	bob   = "b0000000-0000-0000-0000-000000000002"
	carol = "c0000000-0000-0000-0000-000000000003"
)

type fakeDirectory struct {
	profiles  map[string]model.Profile
	err       error
	lastEmail string
}

func (d *fakeDirectory) FindByEmail(_ context.Context, email string) (model.Profile, error) {
	d.lastEmail = email
	if d.err != nil {
		return model.Profile{}, d.err
	}
	for _, p := range d.profiles {
		if p.Email == email {
			return p, nil
		}
	}
	return model.Profile{}, model.ErrNotFound
}

func (d *fakeDirectory) GetByIDs(_ context.Context, ids []string) ([]model.Profile, error) {
	if d.err != nil {
		return nil, d.err
	}
	out := make([]model.Profile, 0, len(ids))
	for _, id := range ids {
		if p, ok := d.profiles[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

type notice struct{ to, from, requestID string }

type fakeNotifier struct {
	requested []notice
	accepted  []notice
	err       error
	called    bool
	ctxErr    error
	deadline  time.Time
	hasLimit  bool
}

func (n *fakeNotifier) record(ctx context.Context) {
	n.called = true
	n.ctxErr = ctx.Err()
	n.deadline, n.hasLimit = ctx.Deadline()
}

func (n *fakeNotifier) FriendRequested(ctx context.Context, to string, from model.Profile, requestID string) error {
	n.record(ctx)
	n.requested = append(n.requested, notice{to, from.ID, requestID})
	return n.err
}

func (n *fakeNotifier) FriendAccepted(ctx context.Context, to string, by model.Profile, requestID string) error {
	n.record(ctx)
	n.accepted = append(n.accepted, notice{to, by.ID, requestID})
	return n.err
}

type fixture struct {
	svc       service.FriendService
	repo      service.FriendshipRepository
	directory *fakeDirectory
	notifier  *fakeNotifier
}

func newFixture() *fixture {
	return newFixtureWithRepo(newMemoryRepo())
}

func newFixtureWithRepo(repo service.FriendshipRepository) *fixture {
	directory := &fakeDirectory{profiles: map[string]model.Profile{
		alice: {ID: alice, Email: "alice@x.io", DisplayName: "Alice"},
		bob:   {ID: bob, Email: "bob@x.io", DisplayName: "Bob"},
		carol: {ID: carol, Email: "carol@x.io", DisplayName: "Carol"},
	}}
	notifier := &fakeNotifier{}
	return &fixture{
		svc:       service.NewFriendService(repo, directory, notifier),
		repo:      repo,
		directory: directory,
		notifier:  notifier,
	}
}

func (f *fixture) invite(t *testing.T, from, to string) string {
	t.Helper()
	if _, err := f.svc.SendRequest(context.Background(), from, to); err != nil {
		t.Fatalf("gửi lời mời: %v", err)
	}
	row, err := f.repo.GetByPair(context.Background(), from, to)
	if err != nil {
		t.Fatalf("đọc lời mời: %v", err)
	}
	return row.ID
}

func (f *fixture) befriend(t *testing.T, a, b string) string {
	t.Helper()
	id := f.invite(t, a, b)
	if err := f.svc.AcceptRequest(context.Background(), b, id); err != nil {
		t.Fatalf("chấp nhận: %v", err)
	}
	return id
}

func requireKind(t *testing.T, err error, kind model.ErrorKind) {
	t.Helper()
	var domainErr *model.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != kind {
		t.Fatalf("phải ra kind %q, nhận %v", kind, err)
	}
}

func TestSendRequestCreatesPendingAndNotifies(t *testing.T) {
	f := newFixture()

	relation, err := f.svc.SendRequest(context.Background(), alice, bob)
	if err != nil {
		t.Fatalf("gửi: %v", err)
	}
	if relation != model.RelationOutgoing {
		t.Fatalf("phải trả outgoing, nhận %q", relation)
	}
	row, err := f.repo.GetByPair(context.Background(), alice, bob)
	if err != nil || row.Status != model.StatusPending {
		t.Fatalf("phải có dòng pending, nhận %+v, %v", row, err)
	}
	want := notice{to: bob, from: alice, requestID: row.ID}
	if len(f.notifier.requested) != 1 || f.notifier.requested[0] != want {
		t.Fatalf("phải báo 1 lần %+v, nhận %+v", want, f.notifier.requested)
	}
}

func TestSendRequestToSelfRejected(t *testing.T) {
	_, err := newFixture().svc.SendRequest(context.Background(), alice, alice)
	requireKind(t, err, model.ErrorKindCannotFriendSelf)
}

func TestSendRequestToDeletedProfileRejected(t *testing.T) {
	f := newFixture()
	delete(f.directory.profiles, bob)

	_, err := f.svc.SendRequest(context.Background(), alice, bob)
	requireKind(t, err, model.ErrorKindProfileNotFound)
}

func TestSendRequestTwiceRejected(t *testing.T) {
	f := newFixture()
	f.invite(t, alice, bob)

	_, err := f.svc.SendRequest(context.Background(), alice, bob)
	requireKind(t, err, model.ErrorKindRequestExists)
}

func TestSendRequestWhenAlreadyFriendsRejected(t *testing.T) {
	f := newFixture()
	f.befriend(t, alice, bob)

	_, err := f.svc.SendRequest(context.Background(), bob, alice)
	requireKind(t, err, model.ErrorKindAlreadyFriends)
}

func TestSendRequestWhenOtherSideInvitedAutoAccepts(t *testing.T) {
	f := newFixture()
	id := f.invite(t, alice, bob)

	relation, err := f.svc.SendRequest(context.Background(), bob, alice)
	if err != nil {
		t.Fatalf("gửi: %v", err)
	}
	if relation != model.RelationFriend {
		t.Fatalf("phải trả friend, nhận %q", relation)
	}
	want := notice{to: alice, from: bob, requestID: id}
	if len(f.notifier.accepted) != 1 || f.notifier.accepted[0] != want {
		t.Fatalf("phải báo chấp nhận %+v, nhận %+v", want, f.notifier.accepted)
	}
}

type racingRepo struct {
	service.FriendshipRepository
	hidden bool
}

func (r *racingRepo) GetByPair(ctx context.Context, a, b string) (model.Friendship, error) {
	if r.hidden {
		r.hidden = false
		return model.Friendship{}, model.ErrNotFound
	}
	return r.FriendshipRepository.GetByPair(ctx, a, b)
}

func TestSendRequestRaceFallsBackToAutoAccept(t *testing.T) {
	inner := newMemoryRepo()
	if _, err := inner.Create(context.Background(), alice, bob); err != nil {
		t.Fatalf("tạo: %v", err)
	}
	f := newFixtureWithRepo(&racingRepo{FriendshipRepository: inner, hidden: true})

	relation, err := f.svc.SendRequest(context.Background(), bob, alice)
	if err != nil {
		t.Fatalf("gửi: %v", err)
	}
	if relation != model.RelationFriend {
		t.Fatalf("phải tự thành bạn, nhận %q", relation)
	}
}

func TestAcceptByAddresseeOnly(t *testing.T) {
	f := newFixture()
	id := f.invite(t, alice, bob)

	requireKind(t, f.svc.AcceptRequest(context.Background(), alice, id), model.ErrorKindRequestNotFound)
	requireKind(t, f.svc.AcceptRequest(context.Background(), carol, id), model.ErrorKindRequestNotFound)

	row, err := f.repo.Get(context.Background(), id)
	if err != nil || row.Status != model.StatusPending {
		t.Fatalf("lời mời phải còn pending, nhận %+v, %v", row, err)
	}
}

func TestAcceptNotifiesRequester(t *testing.T) {
	f := newFixture()
	id := f.befriend(t, alice, bob)

	want := notice{to: alice, from: bob, requestID: id}
	if len(f.notifier.accepted) != 1 || f.notifier.accepted[0] != want {
		t.Fatalf("phải báo %+v, nhận %+v", want, f.notifier.accepted)
	}
}

func TestAcceptTwiceReturnsNotFound(t *testing.T) {
	f := newFixture()
	id := f.befriend(t, alice, bob)

	requireKind(t, f.svc.AcceptRequest(context.Background(), bob, id), model.ErrorKindRequestNotFound)
}

func TestAcceptAfterCancelReturnsNotFound(t *testing.T) {
	f := newFixture()
	id := f.invite(t, alice, bob)
	if err := f.svc.CancelRequest(context.Background(), alice, id); err != nil {
		t.Fatalf("rút: %v", err)
	}

	requireKind(t, f.svc.AcceptRequest(context.Background(), bob, id), model.ErrorKindRequestNotFound)
}

func TestDeclineByAddresseeOnlyDeletesSilently(t *testing.T) {
	f := newFixture()
	id := f.invite(t, alice, bob)

	requireKind(t, f.svc.DeclineRequest(context.Background(), alice, id), model.ErrorKindRequestNotFound)
	if err := f.svc.DeclineRequest(context.Background(), bob, id); err != nil {
		t.Fatalf("từ chối: %v", err)
	}
	if len(f.notifier.accepted) != 0 || len(f.notifier.requested) != 1 {
		t.Fatalf("từ chối không được báo, nhận %+v", f.notifier)
	}
	if _, err := f.svc.SendRequest(context.Background(), alice, bob); err != nil {
		t.Fatalf("phải mời lại được: %v", err)
	}
}

func TestCancelByRequesterOnly(t *testing.T) {
	f := newFixture()
	id := f.invite(t, alice, bob)

	requireKind(t, f.svc.CancelRequest(context.Background(), bob, id), model.ErrorKindRequestNotFound)
	requireKind(t, f.svc.CancelRequest(context.Background(), carol, id), model.ErrorKindRequestNotFound)
	if err := f.svc.CancelRequest(context.Background(), alice, id); err != nil {
		t.Fatalf("rút: %v", err)
	}
	if _, err := f.repo.Get(context.Background(), id); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("dòng phải bị xoá, nhận %v", err)
	}
}

func TestDeclineAndCancelRejectAcceptedFriendship(t *testing.T) {
	f := newFixture()
	id := f.befriend(t, alice, bob)

	requireKind(t, f.svc.DeclineRequest(context.Background(), bob, id), model.ErrorKindRequestNotFound)
	requireKind(t, f.svc.CancelRequest(context.Background(), alice, id), model.ErrorKindRequestNotFound)

	if _, err := f.repo.Get(context.Background(), id); err != nil {
		t.Fatalf("tình bạn phải còn: %v", err)
	}
}

func TestRemoveFriendEitherSideSilently(t *testing.T) {
	for _, tc := range []struct{ name, caller, friend string }{
		{"người gửi hủy", alice, bob},
		{"người nhận hủy", bob, alice},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture()
			f.befriend(t, alice, bob)
			accepted := len(f.notifier.accepted)

			if err := f.svc.RemoveFriend(context.Background(), tc.caller, tc.friend); err != nil {
				t.Fatalf("hủy: %v", err)
			}
			if _, err := f.repo.GetByPair(context.Background(), alice, bob); !errors.Is(err, model.ErrNotFound) {
				t.Fatalf("dòng phải bị xoá, nhận %v", err)
			}
			if len(f.notifier.accepted) != accepted || len(f.notifier.requested) != 1 {
				t.Fatalf("hủy không được báo, nhận %+v", f.notifier)
			}
		})
	}
}

func TestRemoveFriendRejectsStrangerAndPending(t *testing.T) {
	f := newFixture()
	requireKind(t, f.svc.RemoveFriend(context.Background(), alice, carol), model.ErrorKindFriendNotFound)

	f.invite(t, alice, bob)
	requireKind(t, f.svc.RemoveFriend(context.Background(), alice, bob), model.ErrorKindFriendNotFound)
}

func TestSearchNormalizesEmail(t *testing.T) {
	f := newFixture()

	profile, _, err := f.svc.Search(context.Background(), alice, "  BOB@X.io ")
	if err != nil {
		t.Fatalf("tìm: %v", err)
	}
	if f.directory.lastEmail != "bob@x.io" {
		t.Fatalf("email phải được chuẩn hoá, nhận %q", f.directory.lastEmail)
	}
	if profile == nil || profile.ID != bob {
		t.Fatalf("phải thấy bob, nhận %+v", profile)
	}
}

func TestSearchRejectsInvalidEmail(t *testing.T) {
	f := newFixture()
	for _, email := range []string{"", "   ", "abc", "a@b@c", "@x.io", "a@", strings.Repeat("a", 251) + "@x.io"} {
		_, _, err := f.svc.Search(context.Background(), alice, email)
		requireKind(t, err, model.ErrorKindEmailInvalid)
	}
}

func TestSearchNotFoundReturnsNilProfile(t *testing.T) {
	profile, relation, err := newFixture().svc.Search(context.Background(), alice, "nobody@x.io")
	if err != nil {
		t.Fatalf("không thấy không phải lỗi: %v", err)
	}
	if profile != nil || relation != model.RelationNone {
		t.Fatalf("phải trả nil + none, nhận %+v, %q", profile, relation)
	}
}

func TestSearchRelations(t *testing.T) {
	f := newFixture()
	if _, relation, _ := f.svc.Search(context.Background(), alice, "bob@x.io"); relation != model.RelationNone {
		t.Fatalf("chưa quen phải là none, nhận %q", relation)
	}
	if _, relation, _ := f.svc.Search(context.Background(), alice, "alice@x.io"); relation != model.RelationSelf {
		t.Fatalf("chính mình phải là self, nhận %q", relation)
	}

	id := f.invite(t, alice, bob)
	if _, relation, _ := f.svc.Search(context.Background(), alice, "bob@x.io"); relation != model.RelationOutgoing {
		t.Fatalf("đã mời phải là outgoing, nhận %q", relation)
	}
	if _, relation, _ := f.svc.Search(context.Background(), bob, "alice@x.io"); relation != model.RelationIncoming {
		t.Fatalf("được mời phải là incoming, nhận %q", relation)
	}

	if err := f.svc.AcceptRequest(context.Background(), bob, id); err != nil {
		t.Fatalf("chấp nhận: %v", err)
	}
	if _, relation, _ := f.svc.Search(context.Background(), alice, "bob@x.io"); relation != model.RelationFriend {
		t.Fatalf("đã là bạn phải là friend, nhận %q", relation)
	}
}

func TestSearchPropagatesUnavailable(t *testing.T) {
	f := newFixture()
	f.directory.err = model.ErrUnavailable

	_, _, err := f.svc.Search(context.Background(), alice, "bob@x.io")
	if !errors.Is(err, model.ErrUnavailable) {
		t.Fatalf("phải giữ ErrUnavailable, nhận %v", err)
	}
}

func TestSendRequestPropagatesUnavailable(t *testing.T) {
	f := newFixture()
	f.directory.err = model.ErrUnavailable

	_, err := f.svc.SendRequest(context.Background(), alice, bob)
	if !errors.Is(err, model.ErrUnavailable) {
		t.Fatalf("phải giữ ErrUnavailable, nhận %v", err)
	}
}

func TestListFriendsShowsOtherSide(t *testing.T) {
	f := newFixture()
	f.befriend(t, alice, bob)
	f.befriend(t, carol, alice)

	friends, err := f.svc.ListFriends(context.Background(), alice)
	if err != nil {
		t.Fatalf("liệt kê: %v", err)
	}
	seen := map[string]bool{}
	for _, friend := range friends {
		seen[friend.Profile.ID] = true
		if friend.Since.IsZero() {
			t.Fatalf("thiếu mốc kết bạn: %+v", friend)
		}
	}
	if len(friends) != 2 || !seen[bob] || !seen[carol] {
		t.Fatalf("phải thấy bob và carol, nhận %+v", friends)
	}
}

func TestListRequestsSplitsDirections(t *testing.T) {
	f := newFixture()
	sent := f.invite(t, alice, bob)
	received := f.invite(t, carol, alice)

	incoming, outgoing, err := f.svc.ListRequests(context.Background(), alice)
	if err != nil {
		t.Fatalf("liệt kê: %v", err)
	}
	if len(incoming) != 1 || incoming[0].ID != received || incoming[0].Profile.ID != carol {
		t.Fatalf("incoming sai: %+v", incoming)
	}
	if len(outgoing) != 1 || outgoing[0].ID != sent || outgoing[0].Profile.ID != bob {
		t.Fatalf("outgoing sai: %+v", outgoing)
	}
}

func TestListsHideDeletedProfiles(t *testing.T) {
	f := newFixture()
	f.befriend(t, alice, bob)
	f.invite(t, carol, alice)
	delete(f.directory.profiles, bob)
	delete(f.directory.profiles, carol)

	friends, err := f.svc.ListFriends(context.Background(), alice)
	if err != nil || len(friends) != 0 {
		t.Fatalf("phải ẩn bạn đã xoá, nhận %+v, %v", friends, err)
	}
	incoming, _, err := f.svc.ListRequests(context.Background(), alice)
	if err != nil || len(incoming) != 0 {
		t.Fatalf("phải ẩn lời mời của người đã xoá, nhận %+v, %v", incoming, err)
	}
}

func TestNotifierFailureDoesNotFailOperation(t *testing.T) {
	f := newFixture()
	f.notifier.err = errors.New("notification offline")

	id := f.invite(t, alice, bob)
	if err := f.svc.AcceptRequest(context.Background(), bob, id); err != nil {
		t.Fatalf("thông báo lỗi không được làm hỏng thao tác: %v", err)
	}
}

func TestAcceptSucceedsWhenAcceptorProfileLookupFails(t *testing.T) {
	f := newFixture()
	id := f.invite(t, alice, bob)
	f.directory.err = model.ErrUnavailable

	if err := f.svc.AcceptRequest(context.Background(), bob, id); err != nil {
		t.Fatalf("tra tên lỗi không được làm hỏng chấp nhận: %v", err)
	}
	if len(f.notifier.accepted) != 0 {
		t.Fatalf("không có tên thì bỏ qua thông báo, nhận %+v", f.notifier.accepted)
	}
}

func assertShortDetachedContext(t *testing.T, n *fakeNotifier) {
	t.Helper()
	if !n.called {
		t.Fatal("notifier chưa được gọi")
	}
	if n.ctxErr != nil {
		t.Fatalf("thông báo không được phụ thuộc context của request: %v", n.ctxErr)
	}
	if !n.hasLimit || time.Until(n.deadline) > 3*time.Second {
		t.Fatalf("thông báo phải có hạn ngắn, nhận %v, %v", n.deadline, n.hasLimit)
	}
}

func TestSendRequestNotifiesWithShortDetachedContext(t *testing.T) {
	f := newFixture()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := f.svc.SendRequest(ctx, alice, bob); err != nil {
		t.Fatalf("gửi: %v", err)
	}
	assertShortDetachedContext(t, f.notifier)
}

func TestAcceptNotifiesWithShortDetachedContext(t *testing.T) {
	f := newFixture()
	id := f.invite(t, alice, bob)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := f.svc.AcceptRequest(ctx, bob, id); err != nil {
		t.Fatalf("chấp nhận: %v", err)
	}
	assertShortDetachedContext(t, f.notifier)
}
