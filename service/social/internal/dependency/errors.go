package dependency

import (
	"errors"

	"taskmanager/common/errorcode"
	"taskmanager/service/social/internal/model"
)

func ToGRPCError(err error) error {
	var domainErr *model.Error
	if errors.As(err, &domainErr) {
		return errorcode.Error(codeForKind(domainErr.Kind))
	}
	if errors.Is(err, model.ErrUnavailable) {
		return errorcode.Error(errorcode.CommonUnavailable)
	}
	return errorcode.Error(errorcode.CommonInternal)
}

func codeForKind(kind model.ErrorKind) string {
	switch kind {
	case model.ErrorKindCannotFriendSelf:
		return errorcode.SocialCannotFriendSelf
	case model.ErrorKindProfileNotFound:
		return errorcode.SocialProfileNotFound
	case model.ErrorKindRequestExists:
		return errorcode.SocialFriendRequestExists
	case model.ErrorKindAlreadyFriends:
		return errorcode.SocialAlreadyFriends
	case model.ErrorKindRequestNotFound:
		return errorcode.SocialFriendRequestNotFound
	case model.ErrorKindFriendNotFound:
		return errorcode.SocialFriendNotFound
	case model.ErrorKindEmailInvalid:
		return errorcode.SocialEmailInvalid
	default:
		return errorcode.CommonInternal
	}
}
