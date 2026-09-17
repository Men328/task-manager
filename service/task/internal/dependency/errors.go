package dependency

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"taskmanager/service/task/internal/model"
)

func ToGRPCError(err error) error {
	var domainErr *model.Error
	if errors.As(err, &domainErr) {
		return status.Error(errorCode(domainErr.Kind), domainErr.Message)
	}

	switch {
	case errors.Is(err, model.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, model.ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, model.ErrInUse):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, model.ErrCycle):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func errorCode(kind model.ErrorKind) codes.Code {
	switch kind {
	case model.ErrorKindStatusNotInProfile, model.ErrorKindParentNotInProfile:
		return codes.InvalidArgument
	case model.ErrorKindNoStatusAvailable,
		model.ErrorKindStatusProfileMismatch,
		model.ErrorKindTaskStatusSame,
		model.ErrorKindTransitionNotAllowed,
		model.ErrorKindTaskHasSubtasks,
		model.ErrorKindStatusInUse:
		return codes.FailedPrecondition
	default:
		return codes.Internal
	}
}
