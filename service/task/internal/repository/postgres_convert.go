package repository

import "taskmanager/service/task/internal/model"

func priorityToDB(p model.TaskPriority) string {
	switch p {
	case model.TaskPriorityLow:
		return "low"
	case model.TaskPriorityHigh:
		return "high"
	case model.TaskPriorityUrgent:
		return "urgent"
	default:
		return "medium"
	}
}

func priorityFromDB(value string) model.TaskPriority {
	switch value {
	case "low":
		return model.TaskPriorityLow
	case "high":
		return model.TaskPriorityHigh
	case "urgent":
		return model.TaskPriorityUrgent
	default:
		return model.TaskPriorityMedium
	}
}

func priorityToDBPtr(p *model.TaskPriority) any {
	if p == nil {
		return nil
	}
	return priorityToDB(*p)
}

func categoryToDB(c model.TaskStatusCategory) string {
	switch c {
	case model.TaskStatusCategoryInProgress:
		return "in_progress"
	case model.TaskStatusCategoryDone:
		return "done"
	case model.TaskStatusCategoryCancelled:
		return "cancelled"
	default:
		return "todo"
	}
}

func categoryFromDB(value string) model.TaskStatusCategory {
	switch value {
	case "in_progress":
		return model.TaskStatusCategoryInProgress
	case "done":
		return model.TaskStatusCategoryDone
	case "cancelled":
		return model.TaskStatusCategoryCancelled
	default:
		return model.TaskStatusCategoryTodo
	}
}

func categoryToDBPtr(c *model.TaskStatusCategory) any {
	if c == nil {
		return nil
	}
	return categoryToDB(*c)
}

func optionalUUID(id string) any {
	if id == "" {
		return nil
	}
	return id
}
