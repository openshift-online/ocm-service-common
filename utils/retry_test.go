package utils

import (
	"context"
	"fmt"
	"testing"
	"time"

	gm "github.com/onsi/gomega"
)

func TestRetry(t *testing.T) {

	t.Run("Override MaxRetries", func(t *testing.T) {
		gm.RegisterTestingT(t)
		expected := uint64(10)
		r := NewRetry[any]().WithMaxRetries(expected)
		gm.Expect(r.maxRetries).To(gm.Equal(expected))
	})

	t.Run("Override BackoffInterval", func(t *testing.T) {
		gm.RegisterTestingT(t)
		expected := 10 * time.Second
		r := NewRetry[any]().WithConstantBackoff(expected)
		gm.Expect(r.backoffInterval).To(gm.Equal(expected))
	})

	t.Run("Override Context", func(t *testing.T) {
		gm.RegisterTestingT(t)

		//lint:ignore SA1029 the key is not actually used so it's ok to use a string
		expected := context.WithValue(context.Background(), "foo", "bar")
		r := NewRetry[any]().WithContext(expected)
		gm.Expect(r.ctx).To(gm.Equal(expected))
	})

	t.Run("Missing OnEachError callback doesnt break", func(t *testing.T) {
		gm.RegisterTestingT(t)

		opCount := 0
		_, err := NewRetry[any]().
			Do(func() (any, error) {
				if opCount == 0 {
					opCount++
					return nil, fmt.Errorf("Test Error")
				}
				return nil, nil
			}).
			Exec()
		gm.Expect(err).NotTo(gm.HaveOccurred())
	})

	t.Run("OnError function is called MaxRetries times", func(t *testing.T) {
		gm.RegisterTestingT(t)
		maxRetries := 3
		errCount := 0
		_, _ = NewRetry[any]().WithMaxRetries(uint64(maxRetries)).
			OnEachError(func(err error, t time.Duration) {
				errCount++
			}).
			Do(func() (any, error) {
				return nil, fmt.Errorf("Test Error")
			}).
			Exec()
		gm.Expect(errCount).To(gm.Equal(maxRetries))
	})

	t.Run("Operation function is called MaxRetries+1 times (1 intial attempt + MaxRetries)", func(t *testing.T) {
		gm.RegisterTestingT(t)
		maxRetries := 3
		opCount := 0
		_, _ = NewRetry[any]().WithMaxRetries(uint64(maxRetries)).
			Do(func() (any, error) {
				opCount++
				return nil, fmt.Errorf("Test Error")
			}).
			Exec()

		// 1 initial attempt + maxRetries
		gm.Expect(opCount).To(gm.Equal(maxRetries + 1))
	})

	t.Run("Final result is returned", func(t *testing.T) {
		gm.RegisterTestingT(t)

		maxRetries := 3
		opCount := 0
		result, err := NewRetry[any]().
			WithMaxRetries(uint64(maxRetries)).
			Do(func() (any, error) {
				opCount++
				if opCount > maxRetries {
					return "foo", nil
				}
				return nil, fmt.Errorf("Test Error")
			}).
			Exec()
		gm.Expect(result).To(gm.Equal("foo"))
		gm.Expect(err).NotTo(gm.HaveOccurred())
	})

	t.Run("Final error is returned", func(t *testing.T) {
		gm.RegisterTestingT(t)

		maxRetries := 3
		opCount := 0
		_, err := NewRetry[any]().
			WithMaxRetries(uint64(maxRetries)).
			Do(func() (any, error) {
				opCount++
				return nil, fmt.Errorf("Test Error %d", opCount)
			}).
			Exec()
		gm.Expect(err).To(gm.HaveOccurred())

		// 1 initial attempt + maxRetries
		gm.Expect(err.Error()).To(gm.Equal(fmt.Sprintf("Test Error %d", maxRetries+1)))
	})
}
