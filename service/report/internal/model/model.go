package model

import "time"

type StatusCategory string

const (
	CategoryTodo       StatusCategory = "todo"
	CategoryInProgress StatusCategory = "in_progress"
	CategoryDone       StatusCategory = "done"
	CategoryCancelled  StatusCategory = "cancelled"
)

type Status struct {
	ID       string
	Name     string
	Slug     string
	Color    string
	Category StatusCategory
	Position int32
}

type Task struct {
	ID          string
	ParentID    string
	StatusID    string
	Title       string
	Priority    string
	DueAt       *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
	IsArchived  bool
}

type Event struct {
	ID      string
	Status  string
	StartAt time.Time
	EndAt   *time.Time
	AllDay  bool
}

type Schedule struct {
	ID      string
	StartAt time.Time
	EndAt   *time.Time
	AllDay  bool
}

type BacklogItem struct {
	ID        string
	Status    string
	Category  string
	CreatedAt time.Time
}

const (
	EventStatusPlanned   = "planned"
	EventStatusConfirmed = "confirmed"
	EventStatusCancelled = "cancelled"

	BacklogStatusNew      = "NEW"
	BacklogStatusTriaged  = "TRIAGED"
	BacklogStatusArchived = "ARCHIVED"

	BacklogCategoryOther = "other"
)

type Query struct {
	ProfileID       string
	From            *time.Time
	To              *time.Time
	IncludeArchived bool
}

type Scope struct {
	ProfileID string
	From      time.Time
	To        time.Time
}

type Interval string

const (
	IntervalDay   Interval = "day"
	IntervalWeek  Interval = "week"
	IntervalMonth Interval = "month"
)

func (i Interval) Valid() bool {
	switch i {
	case IntervalDay, IntervalWeek, IntervalMonth:
		return true
	default:
		return false
	}
}

type Overview struct {
	ProfileID        string
	From             time.Time
	To               time.Time
	TotalTasks       int32
	RootTasks        int32
	SubtaskCount     int32
	TodoTasks        int32
	InProgressTasks  int32
	DoneTasks        int32
	CancelledTasks   int32
	OverdueTasks     int32
	DueSoonTasks     int32
	CreatedInRange   int32
	CompletedInRange int32
	CompletionRate   float64
	OverdueRate      float64
}

type StatusBreakdownItem struct {
	StatusID   string
	StatusName string
	StatusSlug string
	Color      string
	Category   string
	Count      int32
	Percentage float64
}

type StatusBreakdown struct {
	Items []StatusBreakdownItem
	Total int32
}

type TimeSeriesPoint struct {
	BucketStart time.Time
	Bucket      string
	Created     int32
	Completed   int32
}

type TimeSeries struct {
	Interval       Interval
	Points         []TimeSeriesPoint
	TotalCreated   int32
	TotalCompleted int32
}

type CategoryCount struct {
	Category string
	Count    int32
}

type EventStats struct {
	TotalEvents     int32
	PlannedEvents   int32
	ConfirmedEvents int32
	CancelledEvents int32
	UpcomingEvents  int32
	AllDayEvents    int32
}

type ScheduleStats struct {
	TotalSchedules    int32
	AllDaySchedules   int32
	UpcomingSchedules int32
	TodaySchedules    int32
}

type BacklogStats struct {
	TotalBacklogs    int32
	NewBacklogs      int32
	TriagedBacklogs  int32
	ArchivedBacklogs int32
	ByCategory       []CategoryCount
}

type ActivityReport struct {
	ProfileID string
	From      time.Time
	To        time.Time
	Events    EventStats
	Schedules ScheduleStats
	Backlogs  BacklogStats
}

func (c StatusCategory) Valid() bool {
	switch c {
	case CategoryTodo, CategoryInProgress, CategoryDone, CategoryCancelled:
		return true
	default:
		return false
	}
}
