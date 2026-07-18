package internal

import (
	"path/filepath"
	"sync"
	"testing"
)

// newTestScheduler wires up a DiskManager backed by a temp file plus a
// DiskScheduler on top of it, both cleaned up automatically.
func newTestScheduler(t *testing.T) *DiskScheduler {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sched_test.db")

	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("NewDiskManager failed: %v", err)
	}
	t.Cleanup(func() {
		dm.Close()
	})

	return NewDiskScheduler(dm)
}

// writeSync is a small helper mirroring the ReadPageSync/WritePageSync
// pattern from the checklist -- builds a request, schedules it, blocks on
// the result. Written here directly since you haven't added the caller-side
// helpers to DiskScheduler yet; swap these out once you do.
func writeSync(t *testing.T, s *DiskScheduler, pageID int32, data []byte) {
	t.Helper()
	resultChan := make(chan error, 1)
	s.Schedule(DiskRequest{
		IsWrite:    true,
		PageID:     pageID,
		Data:       data,
		ResultChan: resultChan,
	})
	if err := <-resultChan; err != nil {
		t.Fatalf("write failed: %v", err)
	}
}

func readSync(t *testing.T, s *DiskScheduler, pageID int32) []byte {
	t.Helper()
	buf := make([]byte, PageSize)
	resultChan := make(chan error, 1)
	s.Schedule(DiskRequest{
		IsWrite:    false,
		PageID:     pageID,
		Data:       buf,
		ResultChan: resultChan,
	})
	if err := <-resultChan; err != nil {
		t.Fatalf("read failed: %v", err)
	}
	return buf
}

func TestScheduleWriteThenRead(t *testing.T) {
	s := newTestScheduler(t)

	want := make([]byte, PageSize)
	copy(want, []byte("scheduled hello"))

	writeSync(t, s, 0, want)
	got := readSync(t, s, 0)

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte mismatch at offset %d: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestScheduleSequentialRequestsFromOneGoroutine(t *testing.T) {
	s := newTestScheduler(t)

	for pid := int32(0); pid < 10; pid++ {
		data := make([]byte, PageSize)
		data[0] = byte(pid)
		writeSync(t, s, pid, data)
	}

	for pid := int32(0); pid < 10; pid++ {
		got := readSync(t, s, pid)
		if got[0] != byte(pid) {
			t.Fatalf("page %d: got marker byte %d, want %d", pid, got[0], pid)
		}
	}
}

func TestScheduleConcurrentFromMultipleGoroutines(t *testing.T) {
	// Run with -race to actually catch data races, not just logical bugs.
	s := newTestScheduler(t)

	const numPages = 20
	var wg sync.WaitGroup

	for pid := int32(0); pid < numPages; pid++ {
		wg.Add(1)
		go func(pid int32) {
			defer wg.Done()
			data := make([]byte, PageSize)
			data[0] = byte(pid % 256)
			writeSync(t, s, pid, data)
		}(pid)
	}
	wg.Wait()

	for pid := int32(0); pid < numPages; pid++ {
		got := readSync(t, s, pid)
		want := byte(pid % 256)
		if got[0] != want {
			t.Fatalf("page %d: got marker byte %d, want %d", pid, got[0], want)
		}
	}
}

func TestScheduleReadOfUnwrittenPageReturnsError(t *testing.T) {
	s := newTestScheduler(t)

	buf := make([]byte, PageSize)
	resultChan := make(chan error, 1)
	s.Schedule(DiskRequest{
		IsWrite:    false,
		PageID:     500, // never written
		Data:       buf,
		ResultChan: resultChan,
	})

	if err := <-resultChan; err == nil {
		t.Fatalf("expected error reading unwritten page, got nil")
	}
}

func TestStopDrainsPendingRequests(t *testing.T) {
	s := newTestScheduler(t)

	const numPages = 5
	resultChans := make([]chan error, numPages)

	// Schedule several writes back-to-back, then immediately call Stop.
	// Because Schedule() only enqueues (doesn't wait for completion), all
	// of these can land in the buffer before the worker drains them.
	for pid := int32(0); pid < numPages; pid++ {
		data := make([]byte, PageSize)
		data[0] = byte(pid)
		resultChans[pid] = make(chan error, 1)
		s.Schedule(DiskRequest{
			IsWrite:    true,
			PageID:     pid,
			Data:       data,
			ResultChan: resultChans[pid],
		})
	}

	s.Stop()

	// Every request scheduled before Stop() was called should still
	// complete -- this is the "drain what's queued" behavior.
	for pid := int32(0); pid < numPages; pid++ {
		if err := <-resultChans[pid]; err != nil {
			t.Fatalf("page %d: expected write to complete during drain, got error: %v", pid, err)
		}
	}
}
