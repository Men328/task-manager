package model

import "errors"

var (
	ErrNotFound       = errors.New("subscription not found")
	ErrNotConfigured  = errors.New("mail provider not configured")
	ErrGmailFailed    = errors.New("gmail request failed")
	ErrHistoryGone    = errors.New("gmail history checkpoint expired")
	ErrTokenRefresh   = errors.New("token refresh failed")
	ErrAnalyzeFailed  = errors.New("email analysis failed")
	ErrTaskCreate     = errors.New("task creation failed")
	ErrScheduleCreate = errors.New("schedule creation failed")
	ErrEventCreate    = errors.New("event creation failed")
	ErrBacklogCreate  = errors.New("backlog creation failed")
	ErrStorageFailed  = errors.New("object storage failed")
	ErrQueueFull      = errors.New("notification queue is full")
	ErrUnauthorized   = errors.New("notification is not authenticated")
	ErrInvalidPayload = errors.New("notification payload is invalid")
)
