package audit

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakePublisher struct {
	mu     sync.Mutex
	subs   []Subscriber
	events []*Event
}

func (f *fakePublisher) Publish(ctx context.Context, e *Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	cp := *e
	f.events = append(f.events, &cp)

	return nil
}

func (f *fakePublisher) Subscribe(sub Subscriber) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.subs = append(f.subs, sub)
}

func (f *fakePublisher) Events() []*Event {
	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]*Event(nil), f.events...)
}

func TestAuditService_WorkerPublishesEvent(t *testing.T) {
	pub := &fakePublisher{}

	service := NewAuditService(
		pub,
		slog.Default(),
		10,
		2,
		time.Second,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service.StartAuditWorkerPool(ctx)

	service.Shorten("user1", "https://example.com")

	require.Eventually(t, func() bool {
		return len(pub.Events()) == 1
	}, time.Second, 10*time.Millisecond)

	events := pub.Events()

	require.Equal(t, "shorten", events[0].Action)
	require.Equal(t, "user1", events[0].UserID)
	require.Equal(t, "https://example.com", events[0].URL)
}

func TestAuditService_OverflowQueue(t *testing.T) {
	pub := &fakePublisher{}

	service := NewAuditService(
		pub,
		slog.Default(),
		1,
		1,
		50*time.Millisecond,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service.StartAuditWorkerPool(ctx)

	for i := 0; i < 100; i++ {
		service.Shorten(
			"user",
			"https://example.com",
		)
	}

	require.Eventually(t, func() bool {
		return len(pub.Events()) == 100
	}, 3*time.Second, 20*time.Millisecond)
}

func TestAuditService_Shutdown(t *testing.T) {
	pub := &fakePublisher{}

	service := NewAuditService(
		pub,
		slog.Default(),
		10,
		2,
		time.Second,
	)

	ctx, cancel := context.WithCancel(context.Background())

	service.StartAuditWorkerPool(ctx)

	service.Shorten(
		"user",
		"https://example.com",
	)

	cancel()

	service.Shorten(
		"user1",
		"https://example1.com",
	)

	require.Eventually(t, func() bool {
		return len(pub.Events()) == 2
	}, 2*time.Second, 10*time.Millisecond)
}
