package audit

import (
	"context"
	"errors"
)

type Event struct {
	TS     int64  `json:"ts"`
	Action string `json:"action"`
	UserID string `json:"user_id,omitempty"`
	URL    string `json:"url"`
}

type Subscriber interface {
	Notify(ctx context.Context, event *Event) error
}

type Publisher struct {
	subs []Subscriber
}

func NewPublisher() *Publisher {
	return &Publisher{
		subs: []Subscriber{},
	}
}

func (p *Publisher) Subscribe(sub Subscriber) {
	p.subs = append(p.subs, sub)
}

func (p *Publisher) Publish(ctx context.Context, e *Event) error {
	var errs error

	for _, s := range p.subs {
		err := s.Notify(ctx, e)
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}
