package model

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid argument")
)

type ErrorKind string

const (
	ErrorKindRangeInvalid    ErrorKind = "range_invalid"
	ErrorKindIntervalInvalid ErrorKind = "interval_invalid"
	ErrorKindSourceFailed    ErrorKind = "source_failed"
)

type Error struct {
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string { return e.Message }

func NewError(kind ErrorKind, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}
