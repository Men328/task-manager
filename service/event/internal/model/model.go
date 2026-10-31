package model

import "time"

type Status string

const (
	StatusPlanned   Status = "PLANNED"
	StatusConfirmed Status = "CONFIRMED"
	StatusCancelled Status = "CANCELLED"
)

type Event struct {
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
	Status      Status
	Source      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Filter struct {
	ProfileID string
	ProjectID string
	From      *time.Time
	To        *time.Time
	Status    Status
}

type Update struct {
	ProjectID   *string
	Title       *string
	Description *string
	Location    *string
	StartAt     *time.Time
	EndAt       *time.Time
	AllDay      *bool
	Color       *string
	Status      *Status
}

func (s Status) Valid() bool {
	switch s {
	case StatusPlanned, StatusConfirmed, StatusCancelled:
		return true
	default:
		return false
	}
}
