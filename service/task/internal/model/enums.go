package model

type TaskPriority int32

const (
	TaskPriorityUnspecified TaskPriority = 0
	TaskPriorityLow         TaskPriority = 1
	TaskPriorityMedium      TaskPriority = 2
	TaskPriorityHigh        TaskPriority = 3
	TaskPriorityUrgent      TaskPriority = 4
)

type TaskStatusCategory int32

const (
	TaskStatusCategoryUnspecified TaskStatusCategory = 0
	TaskStatusCategoryTodo        TaskStatusCategory = 1
	TaskStatusCategoryInProgress  TaskStatusCategory = 2
	TaskStatusCategoryDone        TaskStatusCategory = 3
	TaskStatusCategoryCancelled   TaskStatusCategory = 4
)
