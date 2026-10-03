package model

import "time"

type Status string

const (
	StatusNew      Status = "NEW"
	StatusTriaged  Status = "TRIAGED"
	StatusArchived Status = "ARCHIVED"
)

const CategoryOther = "other"

type Item struct {
	ID          string
	ProfileID   string
	Title       string
	Description string
	Sender      string
	Source      string
	Category    string
	Reason      string
	ObjectKey   string
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Filter struct {
	ProfileID string
	Status    Status
	Category  string
}

type Update struct {
	Title       *string
	Description *string
	Reason      *string
	Status      *Status
}

func (s Status) Valid() bool {
	switch s {
	case StatusNew, StatusTriaged, StatusArchived:
		return true
	default:
		return false
	}
}
