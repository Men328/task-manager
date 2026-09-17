package dependency

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/task/internal/model"
)

func ValidateCreateStatusTransition(req *taskv1.CreateStatusTransitionRequest) error {
	if req.GetProfileId() == "" || req.GetFromStatusId() == "" || req.GetToStatusId() == "" {
		return status.Error(codes.InvalidArgument, "profile_id, from_status_id, to_status_id là bắt buộc")
	}
	if req.GetFromStatusId() == req.GetToStatusId() {
		return status.Error(codes.InvalidArgument, "from_status_id và to_status_id không được trùng nhau")
	}
	return nil
}

func ValidateStatusTransitionRequest(req *taskv1.ValidateStatusTransitionRequest) error {
	if req.GetProfileId() == "" || req.GetFromStatusId() == "" || req.GetToStatusId() == "" {
		return status.Error(codes.InvalidArgument, "profile_id, from_status_id, to_status_id là bắt buộc")
	}
	return nil
}

func ValidateStatusTransitionID(id string) error {
	if id == "" {
		return status.Error(codes.InvalidArgument, "id là bắt buộc")
	}
	return nil
}

func TransitionFromCreateRequest(req *taskv1.CreateStatusTransitionRequest) model.Transition {
	return model.Transition{
		ProfileID:    req.GetProfileId(),
		FromStatusID: req.GetFromStatusId(),
		ToStatusID:   req.GetToStatusId(),
		RequiresNote: req.GetRequiresNote(),
		Description:  req.GetDescription(),
	}
}

func TransitionUpdateFromRequest(req *taskv1.UpdateStatusTransitionRequest) model.TransitionUpdate {
	return model.TransitionUpdate{
		IsActive:     req.IsActive,
		RequiresNote: req.RequiresNote,
		Description:  req.Description,
	}
}

func TransitionToProto(t model.Transition) *taskv1.StatusTransition {
	return &taskv1.StatusTransition{
		Id:           t.ID,
		ProfileId:    t.ProfileID,
		FromStatusId: t.FromStatusID,
		ToStatusId:   t.ToStatusID,
		IsActive:     t.IsActive,
		RequiresNote: t.RequiresNote,
		Description:  t.Description,
		CreatedAt:    timestamppb.New(t.CreatedAt),
		UpdatedAt:    timestamppb.New(t.UpdatedAt),
	}
}
