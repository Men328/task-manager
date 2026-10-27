package model

import "time"

type Schedule struct {
	ID          string
	ProfileID   string
	ProjectID   string
	Title       string
	Description string
	Location    string
	StartAt     time.Time
	EndAt       *time.Time
	AllDay      bool
	Color       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ScheduleFilter struct {
	ProfileID string
	ProjectID string
	From      *time.Time
	To        *time.Time
}

type ScheduleUpdate struct {
	ProjectID   *string
	Title       *string
	Description *string
	Location    *string
	StartAt     *time.Time
	EndAt       *time.Time
	AllDay      *bool
	Color       *string
}
