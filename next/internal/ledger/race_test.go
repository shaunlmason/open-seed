package ledger

// The writeHead contention drill (plans/os-c6fb95ee.md): poll-only
// readers Open the store while a supervisor appends — the designed
// multi-process posture — and every Open runs healHead, which
// rewrites a lagging HEAD when it lands in the mid-append window.
// With the shared temp path the reader's rename consumed the
// appender's temp and the append failed with ENOENT
// (TestGracefulPreemptionDrill's CI flake); per-writer unique temps
// make both renames atomic over their own files. The drill hammers
// the mechanism directly: the drill-level window is too narrow to
// hit reliably on fast disks, the store-level one is not.

import (
	"sync"
	"testing"

	"github.com/shaunlmason/open-seed/next/internal/event"
)

func TestConcurrentAppendAndOpenRepair(t *testing.T) {
	priv := fixtureKey(t, 1)
	resolve := fixtureResolver(t, priv)
	dir := t.TempDir()
	s, err := Open(dir, WithClock(clockAt("2026-09-01T10:00:00Z")))
	if err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			// The reader half of the race: each Open runs healHead,
			// which rewrites HEAD when the segments are ahead of it.
			_, _ = Open(dir)
		}
	}()
	tip := event.EmptyHash
	const n = 400
	for i := 0; i < n; i++ {
		rec := signedRecord(t, priv, i, "2026-09-01T10:00:00Z", tip)
		if _, err := s.Append(rec, resolve); err != nil {
			close(stop)
			wg.Wait()
			t.Fatalf("append %d under a concurrent reader: %v", i, err)
		}
		if tip, err = rec.Event.Hash(); err != nil {
			close(stop)
			wg.Wait()
			t.Fatal(err)
		}
	}
	close(stop)
	wg.Wait()
	if rep, err := s.VerifyFromGenesis(resolve); err != nil || rep.Count != n || rep.Tip != tip {
		t.Fatalf("the chain verifies after the contention: %+v %v", rep, err)
	}
}
