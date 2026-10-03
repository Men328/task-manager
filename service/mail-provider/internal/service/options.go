package service

import (
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

type Options struct {
	TopicName           string
	LabelIDs            []string
	RenewInterval       time.Duration
	RenewThreshold      time.Duration
	WorkerCount         int
	DefaultPriority     model.Priority
	AnalyzerEnabled     bool
	RuleFallbackEnabled bool
	ArchiveEnabled      bool
	MaxAttachments      int
	MaxAttachmentBytes  int64
	PullRetryDelay      time.Duration
	PullSubscription    string
	HeartbeatInterval   time.Duration
}

func (o Options) withDefaults() Options {
	out := o
	if out.WorkerCount <= 0 {
		out.WorkerCount = model.DefaultWorkerCount
	}
	if out.RenewThreshold <= 0 {
		out.RenewThreshold = 24 * time.Hour
	}
	if out.PullRetryDelay <= 0 {
		out.PullRetryDelay = 5 * time.Second
	}
	if out.MaxAttachments <= 0 {
		out.MaxAttachments = model.DefaultMaxAttachments
	}
	if out.MaxAttachmentBytes <= 0 {
		out.MaxAttachmentBytes = model.DefaultMaxAttachmentBytes
	}
	return out
}
