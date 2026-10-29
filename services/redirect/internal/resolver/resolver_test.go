// services/redirect/internal/resolver/resolver_test.go
package resolver

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/linkspire/linkspire/services/redirect/internal/model"
)

type fakeL1 struct {
	mu sync.Mutex
	m  map[string]model.Link
}

func newFakeL1() *fakeL1 { return &fakeL1{m: map[string]model.Link{}} }
func (f *fakeL1) Get(k string) (model.Link, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.m[k]
	return v, ok
}
func (f *fakeL1) Add(k string, v model.Link) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.m[k] = v
}

type fakeL2 struct {
	mu       sync.Mutex
	recs     map[string]model.Link
	negative map[string]bool
	setCalls int32
}

func newFakeL2() *fakeL2 {
	return &fakeL2{recs: map[string]model.Link{}, negative: map[string]bool{}}
}
func key(d int64, c string) string { return composeKey(d, c) }
func (f *fakeL2) Get(_ context.Context, d int64, c string) (model.Link, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.recs[key(d, c)]
	return v, ok, nil
}
func (f *fakeL2) Set(_ context.Context, l model.Link, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	atomic.AddInt32(&f.setCalls, 1)
	f.recs[key(l.DomainID, l.Code)] = l
	return nil
}
func (f *fakeL2) IsNegative(_ context.Context, d int64, c string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.negative[key(d, c)], nil
}
func (f *fakeL2) SetNegative(_ context.Context, d int64, c string, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.negative[key(d, c)] = true
	return nil
}

type fakeRepo struct {
	data  map[string]model.Link
	calls int32
	delay time.Duration
}

func (f *fakeRepo) Lookup(_ context.Context, d int64, c string) (model.Link, bool, error) {
	atomic.AddInt32(&f.calls, 1)
	if f.delay > 0 {
		time.Sleep(f.delay)
	}
	v, ok := f.data[key(d, c)]
	return v, ok, nil
}

type fakeBloom struct{ present map[string]bool }

func (f *fakeBloom) MightExist(_ context.Context, d int64, c string) (bool, error) {
	return f.present[key(d, c)], nil
}
func (f *fakeBloom) Add(_ context.Context, d int64, c string) error {
	f.present[key(d, c)] = true
	return nil
}

func activeLink(d int64, c string) model.Link {
	return model.Link{DomainID: d, Code: c, Destination: "https://example.com", Status: model.StatusActive, RedirectType: 302}
}

func TestResolveL1Hit(t *testing.T) {
	l1 := newFakeL1()
	l1.Add(key(1, "abc"), activeLink(1, "abc"))
	r := New(Options{L1: l1})
	got, src, err := r.Resolve(context.Background(), 1, "abc")
	if err != nil || src != SourceL1 || got.Destination != "https://example.com" {
		t.Fatalf("got %+v src=%s err=%v", got, src, err)
	}
}

func TestResolveRedisHitBackfillsL1(t *testing.T) {
	l1 := newFakeL1()
	l2 := newFakeL2()
	_ = l2.Set(context.Background(), activeLink(1, "abc"), 0)
	r := New(Options{L1: l1, L2: l2})
	_, src, err := r.Resolve(context.Background(), 1, "abc")
	if err != nil || src != SourceRedis {
		t.Fatalf("src=%s err=%v", src, err)
	}
	if _, ok := l1.Get(key(1, "abc")); !ok {
		t.Fatal("expected L1 backfill after redis hit")
	}
}

func TestResolvePostgresBackfillsCaches(t *testing.T) {
	l1 := newFakeL1()
	l2 := newFakeL2()
	repo := &fakeRepo{data: map[string]model.Link{key(1, "abc"): activeLink(1, "abc")}}
	// Shared bloom already knows the code exists (maintained by the creation pipeline).
	bloom := &fakeBloom{present: map[string]bool{key(1, "abc"): true}}
	r := New(Options{L1: l1, L2: l2, Repo: repo, Bloom: bloom, RecordTTL: time.Minute})
	_, src, err := r.Resolve(context.Background(), 1, "abc")
	if err != nil || src != SourcePostgres {
		t.Fatalf("src=%s err=%v", src, err)
	}
	if _, ok := l1.Get(key(1, "abc")); !ok {
		t.Fatal("expected L1 backfill")
	}
	if _, ok, _ := l2.Get(context.Background(), 1, "abc"); !ok {
		t.Fatal("expected redis backfill")
	}
	if !bloom.present[key(1, "abc")] {
		t.Fatal("expected bloom add")
	}
}

func TestResolveNotFoundSetsNegative(t *testing.T) {
	l2 := newFakeL2()
	repo := &fakeRepo{data: map[string]model.Link{}}
	r := New(Options{L1: newFakeL1(), L2: l2, Repo: repo, NegativeTTL: time.Minute})
	_, _, err := r.Resolve(context.Background(), 1, "nope")
	if err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if neg, _ := l2.IsNegative(context.Background(), 1, "nope"); !neg {
		t.Fatal("expected negative cache entry")
	}
}

func TestBloomRejectsNonexistent(t *testing.T) {
	repo := &fakeRepo{data: map[string]model.Link{}}
	bloom := &fakeBloom{present: map[string]bool{}} // empty → MightExist=false
	r := New(Options{L1: newFakeL1(), Repo: repo, Bloom: bloom})
	_, _, err := r.Resolve(context.Background(), 1, "ghost")
	if err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if atomic.LoadInt32(&repo.calls) != 0 {
		t.Fatalf("bloom should have short-circuited before repo; repo called %d times", repo.calls)
	}
}

// Stampede protection: many concurrent misses for the same code hit the repo once.
func TestSingleflightCoalesces(t *testing.T) {
	repo := &fakeRepo{data: map[string]model.Link{key(1, "hot"): activeLink(1, "hot")}, delay: 50 * time.Millisecond}
	r := New(Options{L1: newFakeL1(), Repo: repo})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _ = r.Resolve(context.Background(), 1, "hot")
		}()
	}
	wg.Wait()
	if c := atomic.LoadInt32(&repo.calls); c != 1 {
		t.Fatalf("singleflight failed: repo called %d times, want 1", c)
	}
}

func BenchmarkResolveL1Hit(b *testing.B) {
	l1 := newFakeL1()
	l1.Add(key(1, "abc"), activeLink(1, "abc"))
	r := New(Options{L1: l1})
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = r.Resolve(ctx, 1, "abc")
	}
}
