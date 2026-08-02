package stream

import (
	"sync"
	"testing"
	"time"

	"go-judge-system/services/submission/internal/domain/entity"
)

func TestSubmissionEventHubSubscribePublishAndCleanup(t *testing.T) {
	hub := NewSubmissionEventHub().(*SubmissionEventHub)
	events, unsubscribe := hub.Subscribe(77)
	if hub.SubscriberCount(77) != 1 {
		t.Fatalf("subscriber count = %d, want 1", hub.SubscriberCount(77))
	}

	event := entity.SubmissionEvent{SubmissionID: 77, AttemptID: "attempt-1", Status: "JUDGING"}
	hub.Publish(event)

	select {
	case got := <-events:
		if got != event {
			t.Fatalf("event = %+v, want %+v", got, event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	unsubscribe()
	unsubscribe()
	if hub.SubscriberCount(77) != 0 {
		t.Fatalf("subscriber count after unsubscribe = %d, want 0", hub.SubscriberCount(77))
	}
	if _, ok := <-events; ok {
		t.Fatal("events channel is still open after unsubscribe")
	}
}

func TestSubmissionEventHubMultipleSubscribers(t *testing.T) {
	hub := NewSubmissionEventHub().(*SubmissionEventHub)
	first, firstUnsubscribe := hub.Subscribe(77)
	defer firstUnsubscribe()
	second, secondUnsubscribe := hub.Subscribe(77)
	defer secondUnsubscribe()

	event := entity.SubmissionEvent{SubmissionID: 77, AttemptID: "attempt-1", Status: "ACCEPTED"}
	hub.Publish(event)

	for name, ch := range map[string]<-chan entity.SubmissionEvent{"first": first, "second": second} {
		select {
		case got := <-ch:
			if got != event {
				t.Fatalf("%s event = %+v, want %+v", name, got, event)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for %s subscriber", name)
		}
	}
}

func TestSubmissionEventHubSlowSubscriberGetsLatest(t *testing.T) {
	hub := NewSubmissionEventHub().(*SubmissionEventHub)
	events, unsubscribe := hub.Subscribe(77)
	defer unsubscribe()

	oldEvent := entity.SubmissionEvent{SubmissionID: 77, AttemptID: "attempt-1", Status: "JUDGING"}
	latestEvent := entity.SubmissionEvent{SubmissionID: 77, AttemptID: "attempt-1", Status: "ACCEPTED"}
	hub.Publish(oldEvent)
	hub.Publish(latestEvent)

	select {
	case got := <-events:
		if got != latestEvent {
			t.Fatalf("event = %+v, want latest %+v", got, latestEvent)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for latest event")
	}
}

func TestSubmissionEventHubPublishWithoutSubscribersDoesNotBlock(t *testing.T) {
	hub := NewSubmissionEventHub().(*SubmissionEventHub)
	done := make(chan struct{})
	go func() {
		defer close(done)
		hub.Publish(entity.SubmissionEvent{SubmissionID: 404, Status: "ACCEPTED"})
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publish without subscribers blocked")
	}
}

func TestSubmissionEventHubConcurrentPublishUnsubscribe(t *testing.T) {
	hub := NewSubmissionEventHub().(*SubmissionEventHub)
	const workers = 32

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, unsubscribe := hub.Subscribe(77)
			defer unsubscribe()
			for j := 0; j < 100; j++ {
				hub.Publish(entity.SubmissionEvent{
					SubmissionID: 77,
					AttemptID:    "attempt-1",
					Status:       "JUDGING",
				})
			}
		}()
	}
	wg.Wait()

	if hub.SubscriberCount(77) != 0 {
		t.Fatalf("subscriber count after concurrent cleanup = %d, want 0", hub.SubscriberCount(77))
	}
}
