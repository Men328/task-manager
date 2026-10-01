package handler

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mailv1 "taskmanager/common/gen/go/mail/v1"
	"taskmanager/service/mail-provider/internal/model"
)

type stubMailService struct {
	subscribeInput model.SubscribeInput
	subscription   model.Subscription
	accepted       bool
	err            error
}

func (s *stubMailService) Subscribe(_ context.Context, in model.SubscribeInput) (model.Subscription, error) {
	s.subscribeInput = in
	if s.err != nil {
		return model.Subscription{}, s.err
	}
	return s.subscription, nil
}

func (s *stubMailService) Unsubscribe(_ context.Context, _ string) error {
	return s.err
}

func (s *stubMailService) GetSubscription(_ context.Context, _ string) (model.Subscription, error) {
	if s.err != nil {
		return model.Subscription{}, s.err
	}
	return s.subscription, nil
}

func (s *stubMailService) HandleNotification(_ context.Context, _ string, _ string, _ string) (bool, error) {
	return s.accepted, s.err
}

func TestSubscribeValidatesRequiredFields(t *testing.T) {
	handler := NewMailHandler(&stubMailService{})

	cases := []struct {
		name    string
		request *mailv1.SubscribeRequest
	}{
		{name: "thiếu profile", request: &mailv1.SubscribeRequest{Email: "a@b.c", AccessToken: "t"}},
		{name: "thiếu email", request: &mailv1.SubscribeRequest{ProfileId: "p1", AccessToken: "t"}},
		{name: "thiếu access token", request: &mailv1.SubscribeRequest{ProfileId: "p1", Email: "a@b.c"}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := handler.Subscribe(context.Background(), testCase.request)
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("phải là InvalidArgument, nhận %v", err)
			}
		})
	}
}

func TestSubscribeMapsRequestAndResponse(t *testing.T) {
	service := &stubMailService{subscription: model.Subscription{
		ProfileID: "p1",
		Email:     "user@example.com",
		HistoryID: "42",
	}}
	handler := NewMailHandler(service)

	response, err := handler.Subscribe(context.Background(), &mailv1.SubscribeRequest{
		ProfileId:   " p1 ",
		Email:       " user@example.com ",
		AccessToken: " access ",
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if service.subscribeInput.ProfileID != "p1" || service.subscribeInput.AccessToken != "access" {
		t.Fatalf("input phải được trim: %+v", service.subscribeInput)
	}
	if service.subscribeInput.Email != "user@example.com" {
		t.Fatalf("email sai: %q", service.subscribeInput.Email)
	}
	if response.GetSubscription().GetHistoryId() != "42" {
		t.Fatalf("response phải map historyId, nhận %q", response.GetSubscription().GetHistoryId())
	}
}

func TestSubscribeMapsDomainError(t *testing.T) {
	handler := NewMailHandler(&stubMailService{err: model.ErrNotConfigured})

	_, err := handler.Subscribe(context.Background(), &mailv1.SubscribeRequest{
		ProfileId:   "p1",
		Email:       "user@example.com",
		AccessToken: "access",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("ErrNotConfigured phải map thành Internal, nhận %v", err)
	}
}

func TestGetSubscriptionMapsNotFound(t *testing.T) {
	handler := NewMailHandler(&stubMailService{err: model.ErrNotFound})

	_, err := handler.GetSubscription(context.Background(), &mailv1.GetSubscriptionRequest{ProfileId: "p1"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("ErrNotFound phải map thành NotFound, nhận %v", err)
	}
}

func TestGetSubscriptionValidatesProfileID(t *testing.T) {
	handler := NewMailHandler(&stubMailService{})

	_, err := handler.GetSubscription(context.Background(), &mailv1.GetSubscriptionRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("thiếu profile_id phải là InvalidArgument, nhận %v", err)
	}
}

func TestUnsubscribeMapsQueueFullToUnavailable(t *testing.T) {
	handler := NewMailHandler(&stubMailService{err: model.ErrQueueFull})

	_, err := handler.HandleNotification(context.Background(), &mailv1.HandleNotificationRequest{
		Email:     "user@example.com",
		HistoryId: "1",
	})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("ErrQueueFull phải map thành Unavailable, nhận %v", err)
	}
}

func TestHandleNotificationValidatesPayload(t *testing.T) {
	handler := NewMailHandler(&stubMailService{})

	if _, err := handler.HandleNotification(context.Background(), &mailv1.HandleNotificationRequest{Email: "user@example.com"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("thiếu historyId phải là InvalidArgument, nhận %v", err)
	}
	if _, err := handler.HandleNotification(context.Background(), &mailv1.HandleNotificationRequest{HistoryId: "1"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("thiếu email phải là InvalidArgument, nhận %v", err)
	}
}

func TestHandleNotificationReturnsAccepted(t *testing.T) {
	handler := NewMailHandler(&stubMailService{accepted: true})

	response, err := handler.HandleNotification(context.Background(), &mailv1.HandleNotificationRequest{
		Email:     "user@example.com",
		HistoryId: "1",
		MessageId: "m1",
	})
	if err != nil {
		t.Fatalf("handle: %v", err)
	}
	if !response.GetAccepted() {
		t.Fatal("accepted phải được giữ nguyên")
	}
}

func TestUnsubscribeRejectsEmptyProfileID(t *testing.T) {
	handler := NewMailHandler(&stubMailService{})

	_, err := handler.Unsubscribe(context.Background(), &mailv1.UnsubscribeRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("thiếu profile_id phải là InvalidArgument, nhận %v", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatal("không được trả lỗi context")
	}
}
