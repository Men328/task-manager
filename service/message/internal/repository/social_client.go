package repository

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	socialv1 "taskmanager/common/gen/go/social/v1"
	"taskmanager/service/message/internal/model"
)

type SocialClient struct {
	client  socialv1.FriendServiceClient
	timeout time.Duration
}

func NewSocialClient(client socialv1.FriendServiceClient, timeout time.Duration) *SocialClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &SocialClient{client: client, timeout: timeout}
}

func (c *SocialClient) AreFriends(ctx context.Context, profileID, otherID string) (bool, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.AreFriends(callCtx, &socialv1.AreFriendsRequest{ProfileId: profileID, OtherId: otherID})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return false, fmt.Errorf("%w: social AreFriends: %v", model.ErrUnavailable, err)
		default:
			return false, fmt.Errorf("social AreFriends: %w", err)
		}
	}
	return response.GetFriends(), nil
}
