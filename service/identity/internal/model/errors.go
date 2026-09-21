package model

import "errors"

var (
	ErrNotFound             = errors.New("profile not found")
	ErrEmailExists          = errors.New("profile email already exists")
	ErrProfileInactive      = errors.New("profile is inactive")
	ErrAuthProviderNotFound = errors.New("auth provider not found")
)
