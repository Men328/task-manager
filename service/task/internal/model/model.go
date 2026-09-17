package model

import "time"

type Task struct {
	ID           string
	ProfileID    string
	ParentTaskID string
	StatusID     string
	Title        string
	Description  string
	Priority     TaskPriority
	Position     int32
	StartAt      *time.Time
	DueAt        *time.Time
	CompletedAt  *time.Time
	IsArchived   bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type TaskFilter struct {
	ProfileID       string
	ParentTaskID    string
	RootOnly        bool
	StatusID        string
	IncludeArchived bool
}

type TaskUpdate struct {
	ParentTaskID *string
	Title        *string
	Description  *string
	Priority     *TaskPriority
	Position     *int32
	StartAt      *time.Time
	DueAt        *time.Time
	IsArchived   *bool
}

type Status struct {
	ID          string
	ProfileID   string
	Name        string
	Slug        string
	Description string
	Color       string
	Category    TaskStatusCategory
	IsDefault   bool
	IsTerminal  bool
	Position    int32
	IsArchived  bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type StatusUpdate struct {
	Name        *string
	Description *string
	Color       *string
	Category    *TaskStatusCategory
	IsDefault   *bool
	IsTerminal  *bool
	Position    *int32
	IsArchived  *bool
}

type Transition struct {
	ID           string
	ProfileID    string
	FromStatusID string
	ToStatusID   string
	IsActive     bool
	RequiresNote bool
	Description  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type TransitionUpdate struct {
	IsActive     *bool
	RequiresNote *bool
	Description  *string
}

type StatusLog struct {
	ID           int64
	TaskID       string
	ProfileID    string
	FromStatusID string
	ToStatusID   string
	Note         string
	ChangedAt    time.Time
}
