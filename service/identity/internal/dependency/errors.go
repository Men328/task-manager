package dependency

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"taskmanager/service/identity/internal/model"
)

func ToGRPCError(err error) error {
	if errors.Is(err, model.ErrNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}
	return status.Error(codes.Internal, err.Error())
}
