package service

import (
	"context"

	"taskmanager/service/mail-provider/internal/model"
)

type Queue struct {
	ch chan model.Notification
}

func NewQueue(size int) *Queue {
	if size <= 0 {
		size = model.DefaultQueueSize
	}
	return &Queue{ch: make(chan model.Notification, size)}
}

func (q *Queue) Enqueue(notification model.Notification) bool {
	select {
	case q.ch <- notification:
		return true
	default:
		return false
	}
}

func (q *Queue) dequeue(ctx context.Context) (model.Notification, bool) {
	select {
	case <-ctx.Done():
		return model.Notification{}, false
	case notification := <-q.ch:
		return notification, true
	}
}
