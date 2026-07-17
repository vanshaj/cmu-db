package internal

import (
	"testing"
	"time"
)

func newTestReplacer(k int) *LRUKReplacer {
	return NewLRUKReplacer(k)
}

// sleep is used between accesses to guarantee distinct, ordered timestamps.
// time.Now() resolution can be coarse on some systems, so we pad generously.
func tick() {
	time.Sleep(2 * time.Millisecond)
}

func TestEvictSingleEvictableFrame(t *testing.T) {
	r := newTestReplacer(2)

	r.RecordAccess(1)
	r.SetEvictable(1, true)

	frameID, ok := r.Evict()
	if !ok {
		t.Fatalf("expected Evict to succeed")
	}
	if frameID != 1 {
		t.Fatalf("expected frame 1, got %d", frameID)
	}
}

func TestEvictEmptyReplacerReturnsFalse(t *testing.T) {
	r := newTestReplacer(2)

	_, ok := r.Evict()
	if ok {
		t.Fatalf("expected Evict on empty replacer to return ok=false")
	}
}

func TestEvictAllNonEvictableReturnsFalse(t *testing.T) {
	r := newTestReplacer(2)

	r.RecordAccess(1)
	r.SetEvictable(1, false)

	_, ok := r.Evict()
	if ok {
		t.Fatalf("expected Evict to return ok=false when no frames are evictable")
	}
}

func TestInfiniteDistanceBeatsFullHistory(t *testing.T) {
	// K=2. Frame 1 gets 2 accesses (full history, finite K-distance).
	// Frame 2 gets only 1 access (infinite distance).
	// Frame 2 should be evicted first regardless of recency.
	r := newTestReplacer(2)

	r.RecordAccess(1)
	tick()
	r.RecordAccess(1)
	tick()
	r.RecordAccess(2) // only 1 access -> infinite distance

	r.SetEvictable(1, true)
	r.SetEvictable(2, true)

	frameID, ok := r.Evict()
	if !ok {
		t.Fatalf("expected Evict to succeed")
	}
	if frameID != 2 {
		t.Fatalf("expected frame 2 (infinite distance) to be evicted first, got %d", frameID)
	}
}

func TestTiebreakAmongInfiniteDistanceFrames(t *testing.T) {
	// Both frames have < K accesses. The one accessed longer ago
	// (earlier first access) should be evicted first.
	r := newTestReplacer(2)

	r.RecordAccess(1) // accessed first, earliest history
	tick()
	r.RecordAccess(2) // accessed second, more recent history

	r.SetEvictable(1, true)
	r.SetEvictable(2, true)

	frameID, ok := r.Evict()
	if !ok {
		t.Fatalf("expected Evict to succeed")
	}
	if frameID != 1 {
		t.Fatalf("expected frame 1 (earliest access) to be evicted first, got %d", frameID)
	}
}

func TestFullHistoryComparesKthMostRecent(t *testing.T) {
	// Both frames have exactly K=2 accesses. Frame 1's 2nd-most-recent
	// access is older than frame 2's -> frame 1 should be evicted first.
	r := newTestReplacer(2)

	r.RecordAccess(1)
	tick()
	r.RecordAccess(1) // frame 1's K-distance timestamp: this moment
	tick()
	r.RecordAccess(2)
	tick()
	r.RecordAccess(2) // frame 2's K-distance timestamp: this moment (later than frame 1's)

	r.SetEvictable(1, true)
	r.SetEvictable(2, true)

	frameID, ok := r.Evict()
	if !ok {
		t.Fatalf("expected Evict to succeed")
	}
	if frameID != 1 {
		t.Fatalf("expected frame 1 (older K-distance) to be evicted first, got %d", frameID)
	}
}

func TestSetEvictableFalseExcludesFromEviction(t *testing.T) {
	// Frame 1 would normally "win" eviction (infinite distance, oldest),
	// but it's pinned (evictable=false), so frame 2 should be evicted instead.
	r := newTestReplacer(2)

	r.RecordAccess(1)
	tick()
	r.RecordAccess(2)
	tick()
	r.RecordAccess(2)

	r.SetEvictable(1, false) // pinned, should never be picked
	r.SetEvictable(2, true)

	frameID, ok := r.Evict()
	if !ok {
		t.Fatalf("expected Evict to succeed")
	}
	if frameID != 2 {
		t.Fatalf("expected frame 2 (only evictable frame), got %d", frameID)
	}
}

func TestHistoryCappedAtK(t *testing.T) {
	// After more than K accesses, history should still only reflect
	// the K most recent -- verified indirectly via eviction ordering.
	r := newTestReplacer(2)

	r.RecordAccess(1)
	tick()
	r.RecordAccess(1)
	tick()
	r.RecordAccess(1) // 3rd access; oldest should be dropped, history len stays 2
	tick()

	r.RecordAccess(2)
	tick()
	r.RecordAccess(2)

	r.SetEvictable(1, true)
	r.SetEvictable(2, true)

	entry, exists := r.Frames[1]
	if !exists {
		t.Fatalf("expected frame 1 to be tracked")
	}
	if entry.History.Len() != 2 {
		t.Fatalf("expected history capped at K=2, got length %d", entry.History.Len())
	}
}

func TestSizeReflectsOnlyEvictableFrames(t *testing.T) {
	r := newTestReplacer(2)

	r.RecordAccess(1)
	r.RecordAccess(2)
	r.RecordAccess(3)

	r.SetEvictable(1, true)
	r.SetEvictable(2, false)
	r.SetEvictable(3, true)

	if got := r.Size(); got != 2 {
		t.Fatalf("expected Size() == 2, got %d", got)
	}
}

func TestEvictRemovesFrameEntirely(t *testing.T) {
	r := newTestReplacer(2)

	r.RecordAccess(1)
	r.SetEvictable(1, true)

	_, ok := r.Evict()
	if !ok {
		t.Fatalf("expected Evict to succeed")
	}

	if _, exists := r.Frames[1]; exists {
		t.Fatalf("expected frame 1 to be fully removed after eviction")
	}
	if r.Size() != 0 {
		t.Fatalf("expected Size() == 0 after evicting the only frame, got %d", r.Size())
	}
}

func TestRemoveEvictableFrame(t *testing.T) {
	r := newTestReplacer(2)

	r.RecordAccess(1)
	r.SetEvictable(1, true)

	r.Remove(1)

	if _, exists := r.Frames[1]; exists {
		t.Fatalf("expected frame 1 to be removed")
	}
}

func TestSetEvictableOnUntrackedFrame(t *testing.T) {
	// Decide your own behavior here: should this create a new entry,
	// no-op, or panic? This test currently assumes it's safe to call
	// and creates a trackable (but historyless) entry. Adjust to match
	// whatever you implemented.
	r := newTestReplacer(2)

	r.SetEvictable(99, true)

	if got := r.Size(); got != 0 {
		t.Fatalf("expected SetEvictable on untracked frame to register it, Size()==%d", got)
	}
}
