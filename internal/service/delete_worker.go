package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

func (s *URLShortenerService) deleteWorker(ctx context.Context) {

	buff := make([]DeletedTask, 0, s.deleteMaxSize)

	var timer *time.Timer
	var timerC <-chan time.Time

	flush := func(ctx context.Context) {
		if len(buff) == 0 {
			return
		}

		group := make(map[uuid.UUID][]string)

		for _, task := range buff {
			group[task.UID] = append(group[task.UID], task.ShortCodes...)
		}

		for uid, codes := range group {
			if err := s.repo.SoftDeleteURLs(ctx, uid, codes...); err != nil {
				s.log.Error(
					"failed to soft delete URLs",
					"error", err,
				)
			}
		}

		buff = buff[:0]
	}

	for {
		select {
		case <-ctx.Done():

			s.log.Info(
				"delete worker shutting down",
				"pending_tasks", len(buff),
			)

			if timer != nil {
				timer.Stop()
			}

		tail:
			for {
				select {
				case task, ok := <-s.deleteQueue:
					if !ok {
						break tail
					}
					buff = append(buff, task)
				default:
					break tail
				}
			}

			s.log.Info(
				"flushing delete tasks",
				"tasks", len(buff),
			)

			shutDownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			flush(shutDownCtx)
			return
		case task := <-s.deleteQueue:

			buff = append(buff, task)

			if len(buff) == 1 {
				timer = time.NewTimer(s.deleteMaxWait)
				timerC = timer.C
			}

			if len(buff) >= s.deleteMaxSize {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}

				timer = nil
				timerC = nil

				flush(ctx)
			}

		case <-timerC:
			timer = nil
			timerC = nil

			flush(ctx)
		}
	}
}
