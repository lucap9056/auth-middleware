package flight

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type payload struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

func newRedisGroup(t *testing.T, resultTTL time.Duration) *Group {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL not set")
	}
	return mustNew(t, WithRedis(url, resultTTL))
}

func uniqueKey(t *testing.T) string {
	return t.Name() + ":" + time.Now().Format(time.RFC3339Nano)
}

func TestRedisGroup_SharesResultAcrossInstances(t *testing.T) {
	instanceA := newRedisGroup(t, 0)
	instanceB := newRedisGroup(t, 0)
	key := uniqueKey(t)

	var calls atomic.Int32
	fn := func(ctx context.Context) (payload, error) {
		calls.Add(1)
		time.Sleep(200 * time.Millisecond)
		return payload{Value: "shared", Count: 7}, nil
	}

	results := make([]payload, 6)
	errs := make([]error, 6)
	var wg sync.WaitGroup
	for i := range results {
		instance := instanceA
		if i%2 == 1 {
			instance = instanceB
		}
		wg.Go(func() {
			results[i], errs[i] = Do(context.Background(), instance, key, fn)
		})
	}
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Errorf("expected fn to run once across instances, ran %d times", got)
	}
	for i := range results {
		if errs[i] != nil || results[i] != (payload{Value: "shared", Count: 7}) {
			t.Errorf("call %d: got (%+v, %v)", i, results[i], errs[i])
		}
	}
}

func TestRedisGroup_ErrorIsNotCached(t *testing.T) {
	group := newRedisGroup(t, 0)
	key := uniqueKey(t)

	var calls atomic.Int32
	failing := func(ctx context.Context) (payload, error) {
		calls.Add(1)
		return payload{}, context.DeadlineExceeded
	}
	succeeding := func(ctx context.Context) (payload, error) {
		calls.Add(1)
		return payload{Value: "ok"}, nil
	}

	if _, err := Do(context.Background(), group, key, failing); err == nil {
		t.Fatal("expected error from failing fn")
	}
	result, err := Do(context.Background(), group, key, succeeding)
	if err != nil || result.Value != "ok" {
		t.Errorf("got (%+v, %v), want ok result", result, err)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("expected 2 calls, got %d", got)
	}
}

func TestRedisGroup_ResultExpires(t *testing.T) {
	group := newRedisGroup(t, 100*time.Millisecond)
	key := uniqueKey(t)

	var calls atomic.Int32
	fn := func(ctx context.Context) (payload, error) {
		calls.Add(1)
		return payload{Count: int(calls.Load())}, nil
	}

	first, _ := Do(context.Background(), group, key, fn)
	cached, _ := Do(context.Background(), group, key, fn)
	time.Sleep(200 * time.Millisecond)
	expired, _ := Do(context.Background(), group, key, fn)

	if first.Count != 1 || cached.Count != 1 || expired.Count != 2 {
		t.Errorf("got counts (%d, %d, %d), want (1, 1, 2)", first.Count, cached.Count, expired.Count)
	}
}

func TestRedisGroup_InFlightOnlySharesWithConcurrentCallers(t *testing.T) {
	instanceA := newRedisGroup(t, 0)
	instanceB := newRedisGroup(t, 0)
	key := uniqueKey(t)

	var calls atomic.Int32
	fn := func(ctx context.Context) (payload, error) {
		calls.Add(1)
		time.Sleep(200 * time.Millisecond)
		return payload{Value: "in-flight"}, nil
	}

	results := make([]payload, 4)
	var wg sync.WaitGroup
	for i := range results {
		instance := instanceA
		if i%2 == 1 {
			instance = instanceB
		}
		wg.Go(func() {
			results[i], _ = Do(context.Background(), instance, key, fn, InFlightOnly())
		})
	}
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Errorf("expected fn to run once for concurrent callers, ran %d times", got)
	}
	for i, result := range results {
		if result.Value != "in-flight" {
			t.Errorf("call %d: got %+v", i, result)
		}
	}
}

func TestRedisGroup_InFlightOnlyRejectsReplay(t *testing.T) {
	group := newRedisGroup(t, 0)
	key := uniqueKey(t)

	var calls atomic.Int32
	fn := func(ctx context.Context) (payload, error) {
		return payload{Count: int(calls.Add(1))}, nil
	}

	first, _ := Do(context.Background(), group, key, fn, InFlightOnly())
	replay, _ := Do(context.Background(), group, key, fn, InFlightOnly())
	shared, _ := Do(context.Background(), group, key, fn)

	if first.Count != 1 || replay.Count != 2 {
		t.Errorf("got (%d, %d), want replay to rerun as (1, 2)", first.Count, replay.Count)
	}
	if shared.Count != 1 {
		t.Errorf("replay must not overwrite the original result: got %d, want 1", shared.Count)
	}
}
