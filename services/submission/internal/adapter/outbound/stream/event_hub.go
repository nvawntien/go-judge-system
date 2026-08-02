package stream

import (
	"sync"

	"go-judge-system/services/submission/internal/application/port/outbound"
	"go-judge-system/services/submission/internal/domain/entity"
)

const submissionEventBufferSize = 1

type SubmissionEventHub struct {
	mu          sync.RWMutex
	subscribers map[int64]map[*submissionSubscriber]struct{}
}

type submissionSubscriber struct {
	mu     sync.Mutex
	ch     chan entity.SubmissionEvent
	closed bool
}

func NewSubmissionEventHub() outbound.SubmissionEventHub {
	return &SubmissionEventHub{
		subscribers: make(map[int64]map[*submissionSubscriber]struct{}),
	}
}

func (h *SubmissionEventHub) Subscribe(
	submissionID int64,
) (<-chan entity.SubmissionEvent, func()) {
	sub := &submissionSubscriber{
		ch: make(chan entity.SubmissionEvent, submissionEventBufferSize),
	}

	h.mu.Lock()
	if h.subscribers[submissionID] == nil {
		h.subscribers[submissionID] = make(map[*submissionSubscriber]struct{})
	}
	h.subscribers[submissionID][sub] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			if bucket := h.subscribers[submissionID]; bucket != nil {
				delete(bucket, sub)
				if len(bucket) == 0 {
					delete(h.subscribers, submissionID)
				}
			}
			h.mu.Unlock()

			sub.close()
		})
	}

	return sub.ch, unsubscribe
}

func (h *SubmissionEventHub) Publish(event entity.SubmissionEvent) {
	h.mu.RLock()
	subs := make([]*submissionSubscriber, 0, len(h.subscribers[event.SubmissionID]))
	for sub := range h.subscribers[event.SubmissionID] {
		subs = append(subs, sub)
	}
	h.mu.RUnlock()

	for _, sub := range subs {
		sub.trySendLatest(event)
	}
}

func (h *SubmissionEventHub) SubscriberCount(submissionID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers[submissionID])
}

func (s *submissionSubscriber) trySendLatest(event entity.SubmissionEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	select {
	case s.ch <- event:
		return
	default:
	}

	select {
	case <-s.ch:
	default:
	}

	select {
	case s.ch <- event:
	default:
	}
}

func (s *submissionSubscriber) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	close(s.ch)
}
