package handler

import (
	"context"

	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/task/internal/dependency"
	"taskmanager/service/task/internal/service"
)

type TaskHandler struct {
	taskv1.UnimplementedTaskServiceServer
	tasks service.TaskService
}

func NewTaskHandler(tasks service.TaskService) *TaskHandler {
	return &TaskHandler{tasks: tasks}
}

func (h *TaskHandler) CreateTask(ctx context.Context, req *taskv1.CreateTaskRequest) (*taskv1.CreateTaskResponse, error) {
	if err := dependency.ValidateCreateTask(req); err != nil {
		return nil, err
	}

	created, err := h.tasks.Create(ctx, dependency.TaskFromCreateRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &taskv1.CreateTaskResponse{Task: dependency.TaskToProto(created)}, nil
}

func (h *TaskHandler) GetTask(ctx context.Context, req *taskv1.GetTaskRequest) (*taskv1.GetTaskResponse, error) {
	if err := dependency.ValidateTaskID(req.GetId()); err != nil {
		return nil, err
	}

	t, err := h.tasks.Get(ctx, req.GetId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}

	out := dependency.TaskToProto(t)
	if req.GetIncludeSubtasks() {
		children, err := h.tasks.ListChildren(ctx, t.ID)
		if err != nil {
			return nil, dependency.ToGRPCError(err)
		}
		for _, c := range children {
			out.Subtasks = append(out.Subtasks, dependency.TaskToProto(c))
		}
	}
	return &taskv1.GetTaskResponse{Task: out}, nil
}

func (h *TaskHandler) ListTasks(ctx context.Context, req *taskv1.ListTasksRequest) (*taskv1.ListTasksResponse, error) {
	if err := dependency.ValidateListTasks(req); err != nil {
		return nil, err
	}

	items, err := h.tasks.List(ctx, dependency.TaskFilterFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}

	out := make([]*taskv1.Task, 0, len(items))
	for _, t := range items {
		protoTask := dependency.TaskToProto(t)
		if req.GetIncludeSubtasks() {
			children, err := h.tasks.ListChildren(ctx, t.ID)
			if err != nil {
				return nil, dependency.ToGRPCError(err)
			}
			for _, c := range children {
				protoTask.Subtasks = append(protoTask.Subtasks, dependency.TaskToProto(c))
			}
		}
		out = append(out, protoTask)
	}
	return &taskv1.ListTasksResponse{Tasks: out}, nil
}

func (h *TaskHandler) UpdateTask(ctx context.Context, req *taskv1.UpdateTaskRequest) (*taskv1.UpdateTaskResponse, error) {
	if err := dependency.ValidateTaskID(req.GetId()); err != nil {
		return nil, err
	}

	updated, err := h.tasks.Update(ctx, req.GetId(), dependency.TaskUpdateFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &taskv1.UpdateTaskResponse{Task: dependency.TaskToProto(updated)}, nil
}

func (h *TaskHandler) DeleteTask(ctx context.Context, req *taskv1.DeleteTaskRequest) (*taskv1.DeleteTaskResponse, error) {
	if err := dependency.ValidateTaskID(req.GetId()); err != nil {
		return nil, err
	}

	if err := h.tasks.Delete(ctx, req.GetId()); err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &taskv1.DeleteTaskResponse{}, nil
}

func (h *TaskHandler) ChangeTaskStatus(ctx context.Context, req *taskv1.ChangeTaskStatusRequest) (*taskv1.ChangeTaskStatusResponse, error) {
	if err := dependency.ValidateChangeTaskStatus(req); err != nil {
		return nil, err
	}

	updated, err := h.tasks.ChangeStatus(ctx, req.GetId(), req.GetStatusId(), req.GetNote())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &taskv1.ChangeTaskStatusResponse{Task: dependency.TaskToProto(updated)}, nil
}
