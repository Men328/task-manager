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
	ErrorKindNameDuplicate        ErrorKind = "name_duplicate"
	ErrorKindArchived             ErrorKind = "archived"
	ErrorKindDefaultNotArchivable ErrorKind = "default_not_archivable"
	ErrorKindNameTooLong          ErrorKind = "name_too_long"
	ErrorKindColorInvalid         ErrorKind = "color_invalid"
)

type Error struct {
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string { return e.Message }

func NewError(kind ErrorKind, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}
