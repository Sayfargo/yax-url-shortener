package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	mock "github.com/stretchr/testify/mock"
)

func TestDeleteWorker_FlushBySize(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	repo := NewMockURLRepository(t)

	svc := &URLShortenerService{
		repo:          repo,
		deleteQueue:   make(chan DeletedTask, 20),
		log:           log,
		deleteMaxSize: 3,
		deleteMaxWait: time.Hour,
	}

	uid := uuid.New()

	repo.EXPECT().
		SoftDeleteURLs(
			mock.Anything,
			uid,
			[]string{
				"code-0",
				"code-1",
				"code-2",
			},
		).
		Return(nil).
		Once()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go svc.deleteWorker(ctx)

	for i := 0; i < 3; i++ {
		svc.deleteQueue <- DeletedTask{
			UID: uid,
			ShortCodes: []string{
				fmt.Sprintf("code-%d", i),
			},
		}
	}

	time.Sleep(100 * time.Millisecond)

	repo.AssertExpectations(t)
}

func TestDeleteWorker_FlushByTimer(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	repo := NewMockURLRepository(t)

	svc := &URLShortenerService{
		repo:          repo,
		deleteQueue:   make(chan DeletedTask, 10),
		log:           log,
		deleteMaxSize: 100,
		deleteMaxWait: 10 * time.Millisecond,
	}

	uid := uuid.New()

	repo.EXPECT().
		SoftDeleteURLs(
			mock.Anything,
			uid,
			[]string{"abc"},
		).
		Return(nil).
		Once()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go svc.deleteWorker(ctx)

	svc.deleteQueue <- DeletedTask{
		UID: uid,
		ShortCodes: []string{
			"abc",
		},
	}

	time.Sleep(100 * time.Millisecond)

	repo.AssertExpectations(t)
}

func TestDeleteWorker_ShutdownFlush(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	repo := NewMockURLRepository(t)

	svc := &URLShortenerService{
		repo:        repo,
		deleteQueue: make(chan DeletedTask, 10),
		log:         log,

		deleteMaxWait: time.Hour,
		deleteMaxSize: 100,
	}

	uid := uuid.New()

	repo.EXPECT().SoftDeleteURLs(
		mock.Anything,
		uid,
		[]string{"abc"},
	).Return(nil).Once()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		svc.deleteWorker(ctx)
		close(done)
	}()

	svc.deleteQueue <- DeletedTask{
		UID:        uid,
		ShortCodes: []string{"abc"},
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not shutdown")
	}

	repo.AssertExpectations(t)

}
