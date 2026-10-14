package crawler

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type memoryVideoTaskStore struct {
	mu           sync.Mutex
	records      map[string]videoTaskRecord
	retries      map[string]int
	pendingScans atomic.Int64
}

func newMemoryVideoTaskStore() *memoryVideoTaskStore {
	return &memoryVideoTaskStore{records: make(map[string]videoTaskRecord), retries: make(map[string]int)}
}

func (s *memoryVideoTaskStore) put(id, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[id] = videoTaskRecord{Status: status}
}

func (s *memoryVideoTaskStore) lookup(ctx context.Context, id string) (videoTaskRecord, bool, error) {
	if err := ctx.Err(); err != nil {
		return videoTaskRecord{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[id]
	return r, ok, nil
}

func (s *memoryVideoTaskStore) mark(ctx context.Context, id, status string, retries int, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r, exists := s.records[id]
	if exists && r.Status != "resolved" {
		r.Status = status
		s.records[id] = r
		if retries >= 0 {
			s.retries[id] = retries
		}
	}
	return nil
}

func (s *memoryVideoTaskStore) complete(ctx context.Context, id, url, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[id] = videoTaskRecord{Status: "resolved", URL: url, Strategy: "test"}
	return nil
}

func (s *memoryVideoTaskStore) pending(ctx context.Context, cursor string, limit int) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := []string{}
	for id, r := range s.records {
		if id > cursor && (r.Status == "pending" || r.Status == "queued" || r.Status == "resolving" || r.Status == "downloading") {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	if len(ids) > limit {
		ids = ids[:limit]
	}
	s.pendingScans.Add(1)
	return ids, nil
}

func (s *memoryVideoTaskStore) resolvedCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, r := range s.records {
		if r.Status == "resolved" {
			count++
		}
	}
	return count
}

type testVideoResolver struct {
	resolve func(context.Context, string) (string, string, error)
}

func (r testVideoResolver) ResolveWithStrategy(ctx context.Context, id string) (string, string, error) {
	return r.resolve(ctx, id)
}
func (testVideoResolver) Invalidate(string) {}

func testVideoOptions() VideoDownloadOptions {
	o := DefaultVideoDownloadOptions()
	o.TaskTimeout = 3 * time.Second
	o.RetryDelay = time.Millisecond
	o.RefillInterval = time.Minute
	return o
}

func startTestVideoManager(t *testing.T, store *memoryVideoTaskStore, resolver testVideoResolver, options VideoDownloadOptions) *VideoDownloadManager {
	t.Helper()
	m := newVideoDownloadManager(resolver, store)
	if err := m.Start(context.Background(), options); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := m.Shutdown(ctx); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	})
	return m
}

func awaitVideoCondition(t *testing.T, fn func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !fn() {
		if time.Now().After(deadline) {
			t.Fatal(message)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestVideoManagerConstructionAndConfiguration(t *testing.T) {
	store := newMemoryVideoTaskStore()
	var calls atomic.Int32
	m := newVideoDownloadManager(testVideoResolver{func(context.Context, string) (string, string, error) {
		calls.Add(1)
		return "url", "test", nil
	}}, store)
	store.put("123456", "pending")
	if _, err := m.Enqueue(context.Background(), "123456"); !errors.Is(err, ErrVideoManagerStopped) {
		t.Fatalf("enqueue before start: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatal("constructor started work")
	}
	o := testVideoOptions()
	o.Workers = 0
	if err := m.Start(context.Background(), o); err == nil {
		t.Fatal("zero workers accepted")
	}
	o = testVideoOptions()
	o.QueueCapacity = 0
	if err := m.Start(context.Background(), o); err == nil {
		t.Fatal("unbounded/zero queue accepted")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.Start(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("start cancellation: %v", err)
	}
}

func TestVideoManagerSharedLimitAcrossQueueAndRequests(t *testing.T) {
	store := newMemoryVideoTaskStore()
	for i := 0; i < 8; i++ {
		store.put(fmt.Sprint(100000+i), "failed")
	}
	gate := make(chan struct{})
	var running, peak atomic.Int32
	var countMu sync.Mutex
	counts := map[string]int{}
	resolver := testVideoResolver{func(ctx context.Context, id string) (string, string, error) {
		countMu.Lock()
		counts[id]++
		countMu.Unlock()
		n := running.Add(1)
		defer running.Add(-1)
		for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
		}
		select {
		case <-gate:
			return "https://example.test/" + id, "test", nil
		case <-ctx.Done():
			return "", "", ctx.Err()
		}
	}}
	m := startTestVideoManager(t, store, resolver, testVideoOptions())
	for _, id := range []string{"100000", "100001"} {
		if _, err := m.Enqueue(context.Background(), id); err != nil {
			t.Fatal(err)
		}
	}
	awaitVideoCondition(t, func() bool { return running.Load() == 2 }, "queued jobs did not start")
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id string) { defer wg.Done(); _, _, err := m.ResolveNow(context.Background(), id); errs <- err }(fmt.Sprint(100000 + i))
	}
	awaitVideoCondition(t, func() bool { m.mu.Lock(); defer m.mu.Unlock(); return len(m.flights) == 8 }, "requests were not admitted/shared")
	close(gate)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if peak.Load() > 2 {
		t.Fatalf("shared limit exceeded: %d", peak.Load())
	}
	countMu.Lock()
	defer countMu.Unlock()
	for id, n := range counts {
		if n != 1 {
			t.Fatalf("%s resolved %d times", id, n)
		}
	}
}

func TestVideoManagerCallerCancellationKeepsSharedJob(t *testing.T) {
	store := newMemoryVideoTaskStore()
	store.put("123456", "failed")
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	m := startTestVideoManager(t, store, testVideoResolver{func(ctx context.Context, _ string) (string, string, error) {
		calls.Add(1)
		close(started)
		select {
		case <-release:
			return "url", "test", nil
		case <-ctx.Done():
			return "", "", ctx.Err()
		}
	}}, testVideoOptions())
	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)
	go func() { _, _, err := m.ResolveNow(ctx, "123456"); first <- err }()
	<-started
	cancel()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatalf("caller cancellation: %v", err)
	}
	second := make(chan error, 1)
	followerJoined := make(chan struct{})
	followerCtx := &observedVideoWaitContext{Context: context.Background(), joined: followerJoined}
	go func() { _, _, err := m.ResolveNow(followerCtx, "123456"); second <- err }()
	<-followerJoined
	close(release)
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("duplicate execution: %d", calls.Load())
	}
}

func TestVideoManagerAdmissionCancellationAllowsLiveFollower(t *testing.T) {
	store := newMemoryVideoTaskStore()
	gate := make(chan struct{})
	var followCalls atomic.Int32
	o := testVideoOptions()
	o.Workers = 1
	m := startTestVideoManager(t, store, testVideoResolver{func(ctx context.Context, id string) (string, string, error) {
		if id == "100000" {
			select {
			case <-gate:
			case <-ctx.Done():
				return "", "", ctx.Err()
			}
		} else {
			followCalls.Add(1)
		}
		return "url", "test", nil
	}}, o)
	first := make(chan error, 1)
	go func() { _, _, err := m.ResolveNow(context.Background(), "100000"); first <- err }()
	awaitVideoCondition(t, func() bool { return m.Stats().Running == 1 }, "slot not acquired")
	ctx, cancel := context.WithCancel(context.Background())
	leader := make(chan error, 1)
	go func() { _, _, err := m.ResolveNow(ctx, "100001"); leader <- err }()
	awaitVideoCondition(t, func() bool { m.mu.Lock(); defer m.mu.Unlock(); return m.flights["100001"] != nil }, "waiting leader not reserved")
	follower := make(chan error, 1)
	go func() { _, _, err := m.ResolveNow(context.Background(), "100001"); follower <- err }()
	cancel()
	if err := <-leader; !errors.Is(err, context.Canceled) {
		t.Fatalf("leader cancellation: %v", err)
	}
	close(gate)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if err := <-follower; err != nil {
		t.Fatal(err)
	}
	if followCalls.Load() != 1 {
		t.Fatalf("live follower executed %d times", followCalls.Load())
	}
}

func TestVideoManagerFullQueueDefersAndRefills(t *testing.T) {
	store := newMemoryVideoTaskStore()
	for _, id := range []string{"100000", "100001", "100002"} {
		store.put(id, "failed")
	}
	gate, started := make(chan struct{}), make(chan struct{})
	o := testVideoOptions()
	o.Workers = 1
	o.QueueCapacity = 1
	o.RefillInterval = 2 * time.Millisecond
	m := startTestVideoManager(t, store, testVideoResolver{func(ctx context.Context, id string) (string, string, error) {
		if id == "100000" {
			close(started)
			select {
			case <-gate:
			case <-ctx.Done():
				return "", "", ctx.Err()
			}
		}
		return "url", "test", nil
	}}, o)
	if _, err := m.Enqueue(context.Background(), "100000"); err != nil {
		t.Fatal(err)
	}
	<-started
	if _, err := m.Enqueue(context.Background(), "100001"); err != nil {
		t.Fatal(err)
	}
	status, err := m.Enqueue(context.Background(), "100002")
	if err != nil || status != "pending" {
		t.Fatalf("full queue status=%s err=%v", status, err)
	}
	r, _, _ := store.lookup(context.Background(), "100002")
	if r.Status != "pending" {
		t.Fatalf("capacity pressure marked %s", r.Status)
	}
	close(gate)
	awaitVideoCondition(t, func() bool { return store.resolvedCount() == 3 }, "deferred job was not refilled")
}

func TestVideoManagerRecoversMoreThanOneThousandJobs(t *testing.T) {
	store := newMemoryVideoTaskStore()
	for i := 0; i < 1105; i++ {
		store.put(fmt.Sprint(100000+i), "pending")
	}
	o := testVideoOptions()
	o.RefillInterval = 2 * time.Millisecond
	m := startTestVideoManager(t, store, testVideoResolver{func(context.Context, string) (string, string, error) { return "url", "test", nil }}, o)
	awaitVideoCondition(t, func() bool { return store.resolvedCount() == 1105 }, "backlog after first 1000 was not restored")
	awaitVideoCondition(t, func() bool { return m.Stats().Completed == 1105 }, "duplicate or unfinished recovery work")
}

func TestVideoManagerShutdownRestoresInterruptedJobs(t *testing.T) {
	store := newMemoryVideoTaskStore()
	store.put("123456", "failed")
	started := make(chan struct{})
	m := startTestVideoManager(t, store, testVideoResolver{func(ctx context.Context, _ string) (string, string, error) {
		close(started)
		<-ctx.Done()
		return "", "", ctx.Err()
	}}, testVideoOptions())
	result := make(chan error, 1)
	go func() { _, _, err := m.ResolveNow(context.Background(), "123456"); result <- err }()
	<-started
	if err := m.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err == nil {
		t.Fatal("shutdown did not interrupt waiter")
	}
	r, _, _ := store.lookup(context.Background(), "123456")
	if r.Status != "pending" {
		t.Fatalf("interrupted state=%s", r.Status)
	}
	if s := m.Stats(); s.Running != 0 || s.Failed != 0 {
		t.Fatalf("shutdown counted business failure/leaked permit: %+v", s)
	}
	if _, err := m.Enqueue(context.Background(), "123456"); !errors.Is(err, ErrVideoManagerStopped) {
		t.Fatalf("enqueue after shutdown: %v", err)
	}
}

func TestVideoManagerRetryBudgetAndPanicRelease(t *testing.T) {
	store := newMemoryVideoTaskStore()
	store.put("100000", "failed")
	store.put("100001", "failed")
	var calls atomic.Int32
	o := testVideoOptions()
	o.Workers = 1
	m := startTestVideoManager(t, store, testVideoResolver{func(_ context.Context, id string) (string, string, error) {
		if id == "100001" {
			panic("broken resolver")
		}
		calls.Add(1)
		return "", "", errors.New("temporary error")
	}}, o)
	if _, _, err := m.ResolveNow(context.Background(), "100000"); err == nil {
		t.Fatal("failure disappeared")
	}
	if calls.Load() != 3 || m.Stats().Retries != 2 {
		t.Fatalf("retry budget: calls=%d stats=%+v", calls.Load(), m.Stats())
	}
	if _, _, err := m.ResolveNow(context.Background(), "100001"); err == nil || !strings.Contains(err.Error(), "broken resolver") {
		t.Fatalf("panic not reported: %v", err)
	}
	awaitVideoCondition(t, func() bool { return m.Stats().Running == 0 }, "panic leaked permit")
	r, _, _ := store.lookup(context.Background(), "100001")
	if r.Status != "failed" {
		t.Fatalf("panic state=%s", r.Status)
	}
}

func TestVideoManagerTotalTimeoutStopsRetries(t *testing.T) {
	store := newMemoryVideoTaskStore()
	store.put("123456", "failed")
	var calls atomic.Int32
	o := testVideoOptions()
	o.TaskTimeout = 20 * time.Millisecond
	m := startTestVideoManager(t, store, testVideoResolver{func(ctx context.Context, _ string) (string, string, error) {
		calls.Add(1)
		<-ctx.Done()
		return "", "", ctx.Err()
	}}, o)
	if _, _, err := m.ResolveNow(context.Background(), "123456"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("total timeout: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("continued retry after timeout: %d", calls.Load())
	}
	r, _, _ := store.lookup(context.Background(), "123456")
	if r.Status != "failed" {
		t.Fatalf("timeout state=%s", r.Status)
	}
}

type observedVideoWaitContext struct {
	context.Context
	once   sync.Once
	joined chan struct{}
}

func (ctx *observedVideoWaitContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.joined) })
	return ctx.Context.Done()
}

func TestVideoManagerStaleQueueDoesNotRepeatResolvedWork(t *testing.T) {
	store := newMemoryVideoTaskStore()
	store.put("123456", "failed")
	var calls atomic.Int32
	m := startTestVideoManager(t, store, testVideoResolver{func(context.Context, string) (string, string, error) {
		calls.Add(1)
		return "url", "test", nil
	}}, testVideoOptions())
	if _, _, err := m.ResolveNow(context.Background(), "123456"); err != nil {
		t.Fatal(err)
	}
	// Model a queued snapshot which became obsolete after a synchronous resolve.
	m.active.Store("123456", struct{}{})
	m.queue <- "123456"
	awaitVideoCondition(t, func() bool { _, active := m.active.Load("123456"); return !active }, "stale queue entry was not consumed")
	if calls.Load() != 1 || m.Stats().Completed != 1 {
		t.Fatalf("terminal work repeated: calls=%d stats=%+v", calls.Load(), m.Stats())
	}
}

type cleanupPanicVideoStore struct{ *memoryVideoTaskStore }

func (s cleanupPanicVideoStore) mark(ctx context.Context, id, status string, retries int, message string) error {
	if status == "failed" {
		panic("cleanup store panic")
	}
	return s.memoryVideoTaskStore.mark(ctx, id, status, retries, message)
}
func TestVideoManagerCleanupPanicStillReleasesResources(t *testing.T) {
	store := cleanupPanicVideoStore{newMemoryVideoTaskStore()}
	store.put("123456", "failed")
	m := newVideoDownloadManager(testVideoResolver{func(context.Context, string) (string, string, error) { return "", "", errors.New("resolve failed") }}, store)
	opts := testVideoOptions()
	opts.RetryAttempts = 1
	if err := m.Start(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	_, _, err := m.ResolveNow(context.Background(), "123456")
	if err == nil || !strings.Contains(err.Error(), "cleanup store panic") {
		t.Fatalf("cleanup panic: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.Shutdown(ctx); err != nil {
		t.Fatalf("cleanup panic stranded shutdown: %v", err)
	}
	if m.Stats().Running != 0 {
		t.Fatal("cleanup panic leaked execution permit")
	}
}

func TestVideoManagerDoesNotDispatchDiscoveredOnlyRecords(t *testing.T) {
	store := newMemoryVideoTaskStore()
	store.put("123456", "discovered")
	var calls atomic.Int32
	options := testVideoOptions()
	options.RefillInterval = time.Millisecond
	m := startTestVideoManager(t, store, testVideoResolver{func(context.Context, string) (string, string, error) { calls.Add(1); return "url", "test", nil }}, options)
	if result, err := m.ResumePending(context.Background(), 10); err != nil || result.Queued != 0 {
		t.Fatalf("unscheduled discovery queued: %+v %v", result, err)
	}
	if calls.Load() != 0 {
		t.Fatal("discovered-only record parsed automatically")
	}
	if _, err := m.Enqueue(context.Background(), "123456"); err != nil {
		t.Fatal(err)
	}
	awaitVideoCondition(t, func() bool { return store.resolvedCount() == 1 }, "explicit enqueue did not parse discovered video")
}

type onceFailVideoResultStore struct {
	*memoryVideoTaskStore
	failed atomic.Bool
}

func (s *onceFailVideoResultStore) complete(ctx context.Context, id, url, strategy string) error {
	if s.failed.CompareAndSwap(false, true) {
		return errors.New("temporary result persistence failure")
	}
	return s.memoryVideoTaskStore.complete(ctx, id, url, strategy)
}

func TestVideoManagerPersistenceFailureLeavesRecoverableWork(t *testing.T) {
	store := &onceFailVideoResultStore{memoryVideoTaskStore: newMemoryVideoTaskStore()}
	store.put("123456", "failed")
	var calls atomic.Int32
	m := newVideoDownloadManager(testVideoResolver{func(context.Context, string) (string, string, error) {
		calls.Add(1)
		return "url", "test", nil
	}}, store)
	opts := testVideoOptions()
	if err := m.Start(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := m.Shutdown(ctx); err != nil {
			t.Error(err)
		}
	})
	// Join the startup scan before creating new work; this test observes the
	// pending state before explicitly asking for recovery, rather than racing it.
	awaitVideoCondition(t, func() bool { return store.pendingScans.Load() > 0 }, "startup scan did not finish")
	if _, _, err := m.ResolveNow(context.Background(), "123456"); err == nil {
		t.Fatal("result persistence failure was hidden")
	}
	record, _, _ := store.lookup(context.Background(), "123456")
	if record.Status != "pending" || m.Stats().Completed != 0 || m.Stats().Failed != 0 {
		t.Fatalf("persistence error lost work or counted a resolution failure: record=%+v stats=%+v", record, m.Stats())
	}
	if _, err := m.ResumePending(context.Background(), 10); err != nil {
		t.Fatal(err)
	}
	awaitVideoCondition(t, func() bool { return store.resolvedCount() == 1 }, "result persistence failure was not recovered")
	if calls.Load() != 2 {
		t.Fatalf("recovery calls=%d", calls.Load())
	}
}

func TestVideoManagerShutdownInterruptsRetryBackoff(t *testing.T) {
	store := newMemoryVideoTaskStore()
	store.put("123456", "failed")
	var calls atomic.Int32
	m := newVideoDownloadManager(testVideoResolver{func(context.Context, string) (string, string, error) {
		calls.Add(1)
		return "", "", errors.New("retryable error")
	}}, store)
	opts := testVideoOptions()
	opts.RetryDelay = time.Minute
	if err := m.Start(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { _, _, err := m.ResolveNow(context.Background(), "123456"); result <- err }()
	awaitVideoCondition(t, func() bool { return m.Stats().Retries == 1 }, "retry did not enter backoff")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := m.Shutdown(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-result; err == nil {
		t.Fatal("cancelled retry wait reported success")
	}
	if calls.Load() != 1 || m.Stats().Failed != 0 {
		t.Fatalf("shutdown continued/counts failed retries: calls=%d stats=%+v", calls.Load(), m.Stats())
	}
	record, _, _ := store.lookup(context.Background(), "123456")
	if record.Status != "pending" {
		t.Fatalf("interrupted backoff status=%s", record.Status)
	}
}

type blockedTerminalVideoStore struct {
	*memoryVideoTaskStore
	held    atomic.Bool
	joined  chan struct{}
	release chan struct{}
}

func (s *blockedTerminalVideoStore) lookup(ctx context.Context, id string) (videoTaskRecord, bool, error) {
	if s.held.CompareAndSwap(false, true) {
		close(s.joined)
		select {
		case <-s.release:
		case <-ctx.Done():
			return videoTaskRecord{}, false, ctx.Err()
		}
	}
	return s.memoryVideoTaskStore.lookup(ctx, id)
}
func TestVideoManagerSynchronousCallerRetriesSkippedQueueFlight(t *testing.T) {
	base := newMemoryVideoTaskStore()
	base.records["123456"] = videoTaskRecord{Status: "resolved", URL: "old-url", Strategy: "original"}
	store := &blockedTerminalVideoStore{memoryVideoTaskStore: base, joined: make(chan struct{}), release: make(chan struct{})}
	var calls atomic.Int32
	m := newVideoDownloadManager(testVideoResolver{func(context.Context, string) (string, string, error) { calls.Add(1); return "fresh-url", "test", nil }}, store)
	if err := m.Start(context.Background(), testVideoOptions()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := m.Shutdown(ctx); err != nil {
			t.Error(err)
		}
	})
	m.active.Store("123456", struct{}{})
	m.queue <- "123456"
	<-store.joined
	callerJoined := make(chan struct{})
	ctx := &observedVideoWaitContext{Context: context.Background(), joined: callerJoined}
	result := make(chan error, 1)
	go func() {
		url, _, err := m.ResolveNow(ctx, "123456")
		if err == nil && url != "fresh-url" {
			err = fmt.Errorf("returned stale URL %s", url)
		}
		result <- err
	}()
	<-callerJoined
	close(store.release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("synchronous refresh execution count=%d", calls.Load())
	}
}
