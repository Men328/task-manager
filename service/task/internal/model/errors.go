package model

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound = errors.New("not found")

	ErrAlreadyExists = errors.New("already exists")

	ErrInUse = errors.New("in use")

	ErrCycle = errors.New("task cycle detected")
)

type ErrorKind string

const (
	ErrorKindNoStatusAvailable       ErrorKind = "no_status_available"
	ErrorKindStatusNotInProfile      ErrorKind = "status_not_in_profile"
	ErrorKindStatusProfileMismatch   ErrorKind = "status_profile_mismatch"
	ErrorKindParentNotInProfile      ErrorKind = "parent_not_in_profile"
	ErrorKindTaskStatusSame          ErrorKind = "task_status_same"
	ErrorKindTransitionNotAllowed    ErrorKind = "transition_not_allowed"
	ErrorKindTaskHasSubtasks         ErrorKind = "task_has_subtasks"
	ErrorKindStatusInUse             ErrorKind = "status_in_use"
	ErrorKindStatusSlugAlreadyExists ErrorKind = "status_slug_already_exists"
)

type Error struct {
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string { return e.Message }

func NewError(kind ErrorKind, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}
