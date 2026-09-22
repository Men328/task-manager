package model

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound = errors.New("not found")

	ErrAlreadyExists = errors.New("already exists")
)

type ErrorKind string

const (
	ErrorKindSlugAlreadyExists ErrorKind = "slug_already_exists"
)

type Error struct {
	Kind    ErrorKind
	Message string
}

func (e *Error) Error() string { return e.Message }

func NewError(kind ErrorKind, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}
