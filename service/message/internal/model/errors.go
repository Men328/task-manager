package model

import (
	"errors"
	"fmt"
)

var (
	ErrUnavailable = errors.New("upstream unavailable")
	ErrNotFound    = errors.New("not found")
)

type ErrorKind string

const (
	ErrorKindInvalidArgument   ErrorKind = "invalid_argument"
	ErrorKindBodyRequired      ErrorKind = "body_required"
	ErrorKindBodyTooLong       ErrorKind = "body_too_long"
	ErrorKindCannotMessageSelf ErrorKind = "cannot_message_self"
	ErrorKindNotFriends        ErrorKind = "not_friends"
	ErrorKindTaskTitleRequired ErrorKind = "task_title_required"
	ErrorKindMessageNotFound   ErrorKind = "message_not_found"
)

type Error struct {
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string { return e.Message }

func NewError(kind ErrorKind, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}
