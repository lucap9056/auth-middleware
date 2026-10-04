package flight

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func mustNew(t *testing.T, opts ...Option) *Group {
	t.Helper()
	group, err := New(opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(group.Close)
	return group
}

func runConcurrently[T any](t *testing.T, group *Group, n int, key string, fn func(ctx context.Context) (T, error)) ([]T, []error) {
	t.Helper()
	results := make([]T, n)
	errs := make([]error, n)

	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			results[i], errs[i] = Do(context.Background(), group, key, fn)
		})
	}
	wg.Wait()
	return results, errs
}

func TestLocalGroup_DeduplicatesConcurrentCalls(t *testing.T) {
	group := mustNew(t)

	var calls atomic.Int32
	results, errs := runConcurrently(t, group, 10, "key", func(ctx context.Context) (int, error) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		return 42, nil
	})

	if got := calls.Load(); got != 1 {
		t.Errorf("expected fn to run once, ran %d times", got)
	}
	for i := range results {
		if errs[i] != nil || results[i] != 42 {
			t.Errorf("call %d: got (%d, %v), want (42, nil)", i, results[i], errs[i])
		}
	}
}

func TestLocalGroup_DifferentKeysRunSeparately(t *testing.T) {
	group := mustNew(t)

	a, _ := Do(context.Background(), group, "a", func(ctx context.Context) (string, error) { return "A", nil })
	b, _ := Do(context.Background(), group, "b", func(ctx context.Context) (string, error) { return "B", nil })

	if a != "A" || b != "B" {
		t.Errorf("got (%q, %q), want (\"A\", \"B\")", a, b)
	}
}

func TestLocalGroup_PropagatesError(t *testing.T) {
	group := mustNew(t)
	wantErr := errors.New("boom")

	result, err := Do(context.Background(), group, "key", func(ctx context.Context) (*struct{}, error) {
		return nil, wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Errorf("expected %v, got %v", wantErr, err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestLocalGroup_NilInterfaceResult(t *testing.T) {
	group := mustNew(t)

	result, err := Do(context.Background(), group, "key", func(ctx context.Context) (error, error) {
		return nil, nil
	})

	if result != nil || err != nil {
		t.Errorf("got (%v, %v), want (nil, nil)", result, err)
	}
}

func TestGroup_SharedAcrossResultTypes(t *testing.T) {
	group := mustNew(t)

	count, err := Do(context.Background(), group, "count", func(ctx context.Context) (int, error) { return 3, nil })
	if err != nil || count != 3 {
		t.Errorf("int result: got (%d, %v), want (3, nil)", count, err)
	}

	name, err := Do(context.Background(), group, "name", func(ctx context.Context) (string, error) { return "flight", nil })
	if err != nil || name != "flight" {
		t.Errorf("string result: got (%q, %v), want (\"flight\", nil)", name, err)
	}
}

func TestLocalGroup_CachesResultUntilExpired(t *testing.T) {
	group := mustNew(t, WithRedis("", 100*time.Millisecond))

	var calls atomic.Int32
	fn := func(ctx context.Context) (int, error) {
		return int(calls.Add(1)), nil
	}

	first, _ := Do(context.Background(), group, "key", fn)
	cached, _ := Do(context.Background(), group, "key", fn)
	time.Sleep(200 * time.Millisecond)
	expired, _ := Do(context.Background(), group, "key", fn)

	if first != 1 || cached != 1 || expired != 2 {
		t.Errorf("got (%d, %d, %d), want (1, 1, 2)", first, cached, expired)
	}
}

func TestLocalGroup_ErrorIsNotCached(t *testing.T) {
	group := mustNew(t)

	var calls atomic.Int32
	failing := func(ctx context.Context) (int, error) {
		calls.Add(1)
		return 0, errors.New("boom")
	}

	Do(context.Background(), group, "key", failing)
	Do(context.Background(), group, "key", failing)

	if got := calls.Load(); got != 2 {
		t.Errorf("expected failed calls to rerun, ran %d times", got)
	}
}

func TestLocalGroup_InFlightOnlyDoesNotReuseFinishedResult(t *testing.T) {
	group := mustNew(t)

	var calls atomic.Int32
	fn := func(ctx context.Context) (int, error) {
		return int(calls.Add(1)), nil
	}

	first, _ := Do(context.Background(), group, "key", fn, InFlightOnly())
	replay, _ := Do(context.Background(), group, "key", fn, InFlightOnly())

	if first != 1 || replay != 2 {
		t.Errorf("got (%d, %d), want (1, 2)", first, replay)
	}
}
