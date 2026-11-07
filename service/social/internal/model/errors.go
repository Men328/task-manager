package model

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrDuplicate   = errors.New("friendship already exists")
	ErrUnavailable = errors.New("upstream unavailable")
)

type ErrorKind string

const (
	ErrorKindCannotFriendSelf ErrorKind = "cannot_friend_self"
	ErrorKindProfileNotFound  ErrorKind = "profile_not_found"
	ErrorKindRequestExists    ErrorKind = "request_exists"
	ErrorKindAlreadyFriends   ErrorKind = "already_friends"
	ErrorKindRequestNotFound  ErrorKind = "request_not_found"
	ErrorKindFriendNotFound   ErrorKind = "friend_not_found"
	ErrorKindEmailInvalid     ErrorKind = "email_invalid"
)

type Error struct {
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string { return e.Message }

func NewError(kind ErrorKind, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}
