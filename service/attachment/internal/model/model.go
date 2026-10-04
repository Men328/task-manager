package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrInvalid       = errors.New("invalid argument")
	ErrTooLarge      = errors.New("attachment too large")
	ErrStorageFailed = errors.New("object storage failed")
)

type ErrorKind string

const (
	ErrorKindOwnerTypeInvalid ErrorKind = "owner_type_invalid"
	ErrorKindFileTooLarge     ErrorKind = "file_too_large"
	ErrorKindFileRequired     ErrorKind = "file_required"
	ErrorKindStorageFailed    ErrorKind = "storage_failed"
)

type Error struct {
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string { return e.Message }

func NewError(kind ErrorKind, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}

type Attachment struct {
	ID          string
	ProfileID   string
	OwnerType   string
	OwnerID     string
	FileName    string
	ContentType string
	Size        int64
	ObjectKey   string
	CreatedAt   time.Time
}

type Upload struct {
	ProfileID   string
	OwnerType   string
	OwnerID     string
	FileName    string
	ContentType string
	Content     []byte
}

type Filter struct {
	ProfileID string
	OwnerType string
	OwnerID   string
}

const (
	OwnerTypeTask     = "task"
	OwnerTypeSchedule = "schedule"
	OwnerTypeEvent    = "event"
	OwnerTypeBacklog  = "backlog"
)

var OwnerTypes = []string{OwnerTypeTask, OwnerTypeSchedule, OwnerTypeEvent, OwnerTypeBacklog}

func NormalizeOwnerType(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case OwnerTypeTask:
		return OwnerTypeTask, true
	case OwnerTypeSchedule:
		return OwnerTypeSchedule, true
	case OwnerTypeEvent:
		return OwnerTypeEvent, true
	case OwnerTypeBacklog:
		return OwnerTypeBacklog, true
	default:
		return "", false
	}
}
