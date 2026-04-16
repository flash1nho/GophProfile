package worker

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeTimer struct{}

func (f fakeTimer) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	ch <- time.Now()
	return ch
}

func TestRetry_SuccessFirstTry(t *testing.T) {
	ctx := context.Background()

	called := 0

	err := retry(ctx, 3, time.Second, fakeTimer{}, func() error {
		called++
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if called != 1 {
		t.Fatalf("expected 1 call, got %d", called)
	}
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()

	called := 0

	err := retry(ctx, 5, time.Second, fakeTimer{}, func() error {
		called++
		if called < 3 {
			return errors.New("fail")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if called != 3 {
		t.Fatalf("expected 3 calls, got %d", called)
	}
}

func TestRetry_AllFail(t *testing.T) {
	ctx := context.Background()

	called := 0

	err := retry(ctx, 3, time.Second, fakeTimer{}, func() error {
		called++
		return errors.New("fail")
	})

	if err == nil {
		t.Fatalf("expected error")
	}

	if called != 3 {
		t.Fatalf("expected 3 calls, got %d", called)
	}
}

func TestRetry_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	called := 0

	err := retry(ctx, 5, time.Second, fakeTimer{}, func() error {
		called++
		cancel()
		return errors.New("fail")
	})

	if err == nil {
		t.Fatalf("expected error")
	}

	if called != 1 {
		t.Fatalf("expected 1 call, got %d", called)
	}
}
