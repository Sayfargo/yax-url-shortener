package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sayfargo/yax-url-shortener/internal/model"
	"github.com/google/uuid"
	mock "github.com/stretchr/testify/mock"
)

type MockURLRepo struct {
	deleted atomic.Int64
	calls   atomic.Int64
	done    chan struct{}
	target  int64
}

func (mr *MockURLRepo) Create(ctx context.Context, shortenedURL model.ShortenedURL) error {
	return nil
}

func (mr *MockURLRepo) CreateBatch(ctx context.Context, shortenedURLs []model.ShortenedURL) error {
	return nil
}

func (mr *MockURLRepo) Get(ctx context.Context, shortCode string) (string, error) {
	return "", nil
}

func (mr *MockURLRepo) GetURLs(ctx context.Context, uid uuid.UUID) ([]model.ShortenedURL, error) {
	return nil, nil
}

func (mr *MockURLRepo) SoftDeleteURLs(ctx context.Context, uid uuid.UUID, shortCodes ...string) error {
	mr.calls.Add(1)
	if mr.deleted.Add(int64(len(shortCodes))) >= mr.target {
		select {
		case mr.done <- struct{}{}:
		default:
		}
	}
	time.Sleep(3 * time.Millisecond) // работа БД
	return nil
}

func BenchmarkDeleteWorker(b *testing.B) {
	log := slog.New(slog.DiscardHandler)

	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	randStr := func(size int) string {
		b := make([]byte, size)
		for i := range b {
			b[i] = letterBytes[rand.IntN(len(letterBytes))]
		}
		return string(b)
	}

	tasks := make([]DeletedTask, 0, 10000)
	for range cap(tasks) {
		shortCodes := make([]string, 0, rand.IntN(11)+2)
		for range cap(shortCodes) {
			shortCodes = append(shortCodes, randStr(8))
		}
		tasks = append(tasks, DeletedTask{
			UID:        uuid.New(),
			ShortCodes: shortCodes,
		})
	}

	var total int64
	for i := 0; i < b.N; i++ {
		total += int64(len(tasks[i%len(tasks)].ShortCodes))
	}

	mockRepo := &MockURLRepo{done: make(chan struct{}, 1), target: total}
	const maxBatchSize = 10
	svc := &URLShortenerService{
		repo:          mockRepo,
		deleteQueue:   make(chan DeletedTask, maxBatchSize),
		log:           log,
		deleteMaxSize: maxBatchSize,
		deleteMaxWait: time.Millisecond * 5,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go svc.deleteWorker(ctx)

	b.ReportAllocs()
	b.ResetTimer()

	idx := 0
	for b.Loop() {
		svc.deleteQueue <- tasks[idx%len(tasks)]
		idx++
	}

	select {
	case <-mockRepo.done:
	case <-time.After(10 * time.Second):
		b.Fatalf("worker processed %d of %d codes", mockRepo.deleted.Load(), total)
	}
	b.StopTimer()

	b.ReportMetric(float64(mockRepo.calls.Load())/float64(b.N), "repo-calls/op")
	b.ReportMetric(float64(mockRepo.deleted.Load())/float64(b.N), "codes/op")
}

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
