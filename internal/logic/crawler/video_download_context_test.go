package crawler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Redirect the resolver's platform request to a local server without changing
// its request context or invoking a browser/network service in these tests.
type videoResolverTestTransport struct {
	base      *url.URL
	transport http.RoundTripper
}

func (r videoResolverTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	local := req.Clone(req.Context())
	local.URL.Scheme = r.base.Scheme
	local.URL.Host = r.base.Host
	return r.transport.RoundTrip(local)
}

func waitVideoResolverTestResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("video resolution did not stop after cancellation")
		return nil
	}
}

func waitVideoResolverTestRequest(t *testing.T, started <-chan struct{}) {
	t.Helper()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("local request did not start")
	}
}

func TestVerifyDownloadURLCancellationInterruptsBodyRead(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.WriteHeader(http.StatusPartialContent)
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	d := NewVideoDownloader()
	go func() {
		_, err := d.verifyDownloadURL(ctx, server.URL+"/media.mp4")
		result <- err
	}()
	waitVideoResolverTestRequest(t, started)
	cancel()
	if err := waitVideoResolverTestResult(t, result); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestFinalizeDownloadCandidatesStopsAfterCancellation(t *testing.T) {
	started := make(chan struct{})
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			close(started)
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	d := NewVideoDownloader()
	go func() {
		_, err := d.finalizeDownloadURLCandidates(ctx, []string{
			server.URL + "/playwm/first.mp4", // contains an additional watermark fallback
			server.URL + "/second.mp4",
		})
		result <- err
	}()
	waitVideoResolverTestRequest(t, started)
	cancel()
	if err := waitVideoResolverTestResult(t, result); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("issued %d requests after cancellation, want 1 total", got)
	}
}

func TestResolveCancellationDoesNotFallBackToBrowser(t *testing.T) {
	started := make(chan struct{})
	var startedOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedOnce.Do(func() { close(started) })
		<-r.Context().Done()
	}))
	defer server.Close()
	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	d := NewVideoDownloader()
	d.client.Transport = videoResolverTestTransport{base: base, transport: server.Client().Transport}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, _, err := d.ResolveWithStrategy(ctx, "123456789")
		result <- err
	}()
	waitVideoResolverTestRequest(t, started)
	cancel()
	if err := waitVideoResolverTestResult(t, result); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled without a browser fallback", err)
	}
}

func TestResolveCancelledContextDoesNotReturnCache(t *testing.T) {
	d := NewVideoDownloader()
	d.setCache("cached", "https://example.com/cached.mp4")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, strategy, err := d.ResolveWithStrategy(ctx, "cached"); got != "" || strategy != "" || !errors.Is(err, context.Canceled) {
		t.Fatalf("ResolveWithStrategy returned %q, %q, %v", got, strategy, err)
	}
	if got, err := d.resolveByDesktopBrowser(ctx, "cancelled", false); got != "" || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled browser resolution returned %q, %v", got, err)
	}
}

func TestCollectVideoCandidatesCancellationAndFiltering(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := collectVideoCandidates(ctx, make(chan string), time.Hour)
		result <- err
	}()
	cancel()
	if err := waitVideoResolverTestResult(t, result); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}

	candidates := make(chan string, 4)
	candidates <- " https://example.com/first.mp4 "
	candidates <- "https://example.com/first.mp4"
	candidates <- "https://example.com/douyin-pc-web/uuu_265.mp4"
	candidates <- "https://example.com/second.mp4"
	close(candidates)
	got, err := collectVideoCandidates(context.Background(), candidates, time.Hour)
	want := []string{"https://example.com/first.mp4", "https://example.com/second.mp4"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates = %v, error = %v, want %v", got, err, want)
	}
}

func TestFinalizeDownloadCandidatesKeepsVideoFallback(t *testing.T) {
	var paths []string
	var pathsMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathsMu.Lock()
		paths = append(paths, r.URL.Path)
		pathsMu.Unlock()
		if r.Header.Get("Range") != "bytes=0-1023" {
			t.Errorf("Range header = %q", r.Header.Get("Range"))
		}
		if r.URL.Path != "/valid.mp4" {
			http.Error(w, "candidate is unavailable", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "video/mp4")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = fmt.Fprint(w, "video prefix")
	}))
	defer server.Close()
	d := NewVideoDownloader()
	got, err := d.finalizeDownloadURLCandidates(context.Background(), []string{
		server.URL + "/playwm/first.mp4", server.URL + "/valid.mp4",
	})
	if err != nil || got != server.URL+"/valid.mp4" {
		t.Fatalf("url = %q, error = %v", got, err)
	}
	wantPaths := []string{"/play/first.mp4", "/playwm/first.mp4", "/valid.mp4"}
	pathsMu.Lock()
	defer pathsMu.Unlock()
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("request paths = %v, want %v", paths, wantPaths)
	}
}
