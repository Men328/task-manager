package model

import "time"

type Workspace struct {
	ID             string
	OwnerProfileID string
	Name           string
	Slug           string
	Description    string
	Color          string
	Icon           string
	IsDefault      bool
	Position       int32
	IsArchived     bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

type WorkspaceFilter struct {
	OwnerProfileID  string
	IncludeArchived bool
}

type WorkspaceUpdate struct {
	Name        *string
	Description *string
	Color       *string
	Icon        *string
	IsDefault   *bool
	Position    *int32
	IsArchived  *bool
}
