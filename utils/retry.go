package utils

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
)

type Retry[T any] struct {
	ctx             context.Context
	maxRetries      uint64
	backoffInterval time.Duration
	operation       func() (T, error)
	onError         func(err error, t time.Duration)
}

func NewRetry[T any]() *Retry[T] {
	return &Retry[T]{
		ctx:             context.Background(),
		maxRetries:      3,
		backoffInterval: 10 * time.Millisecond,
	}
}

func (r *Retry[T]) WithContext(ctx context.Context) *Retry[T] {
	r.ctx = ctx
	return r
}

func (r *Retry[T]) WithMaxRetries(maxRetries uint64) *Retry[T] {
	r.maxRetries = maxRetries
	return r
}

func (r *Retry[T]) WithConstantBackoff(backoffInterval time.Duration) *Retry[T] {
	r.backoffInterval = backoffInterval
	return r
}

func (r *Retry[T]) Do(operation func() (T, error)) *Retry[T] {
	r.operation = operation
	return r
}

func (r *Retry[T]) OnEachError(onError func(err error, t time.Duration)) *Retry[T] {
	r.onError = onError
	return r
}

func (r *Retry[T]) Exec() (T, error) {
	var b backoff.BackOff

	b = backoff.NewConstantBackOff(r.backoffInterval)
	b = backoff.WithContext(b, r.ctx)
	b = backoff.WithMaxRetries(b, r.maxRetries)

	var result T
	opWrapper := func() error {
		var err error
		result, err = r.operation()
		return err
	}

	err := backoff.RetryNotify(opWrapper, b, r.onError)
	return result, err
}
