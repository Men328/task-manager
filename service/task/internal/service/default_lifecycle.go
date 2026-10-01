package service

import "taskmanager/service/task/internal/model"

type defaultStatusDef struct {
	Name       string
	Slug       string
	Color      string
	Category   model.TaskStatusCategory
	IsDefault  bool
	IsTerminal bool
	Position   int32
}

var seedDefaultStatuses = []defaultStatusDef{
	{Name: "To Do", Slug: "todo", Color: "#9aa0ae", Category: model.TaskStatusCategoryTodo, IsDefault: true, Position: 0},
	{Name: "On Progress", Slug: "on-progress", Color: "#f59f00", Category: model.TaskStatusCategoryInProgress, Position: 1},
	{Name: "In Review", Slug: "in-review", Color: "#37b24d", Category: model.TaskStatusCategoryInProgress, Position: 2},
	{Name: "Completed", Slug: "completed", Color: "#f06595", Category: model.TaskStatusCategoryDone, IsTerminal: true, Position: 3},
}

var seedDefaultTransitionPaths = [][2]string{
	{"todo", "on-progress"},
	{"on-progress", "in-review"},
	{"in-review", "completed"},
	{"in-review", "on-progress"},
	{"completed", "on-progress"},
}
