package model

import "time"

// DefaultProjectName là tên project mặc định của mỗi profile.
const DefaultProjectName = "Chung"

type Project struct {
	ID          string
	ProfileID   string
	Name        string
	Description string
	Color       string
	IsDefault   bool
	IsArchived  bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ProjectUpdate struct {
	Name        *string
	Description *string
	Color       *string
}

type ProjectFilter struct {
	ProfileID       string
	IncludeArchived bool
}
