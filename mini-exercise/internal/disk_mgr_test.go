package internal

import (
	"os"
	"path/filepath"
	"testing"
)

// newTestDiskManager creates a DiskManager backed by a temp file that is
// automatically cleaned up when the test ends.
func newTestDiskManager(t *testing.T) (*DiskManager, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	dm, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("NewDiskManager failed: %v", err)
	}
	t.Cleanup(func() {
		dm.Close()
	})
	return dm, path
}

func TestWriteReadRoundTrip(t *testing.T) {
	dm, _ := newTestDiskManager(t)

	pageID := dm.AllocatePage()

	want := make([]byte, PageSize)
	copy(want, []byte("hello disk manager"))

	if err := dm.WritePage(pageID, want); err != nil {
		t.Fatalf("WritePage failed: %v", err)
	}

	got, err := dm.ReadPage(pageID)
	if err != nil {
		t.Fatalf("ReadPage failed: %v", err)
	}

	if len(got) != PageSize {
		t.Fatalf("ReadPage returned %d bytes, want %d", len(got), PageSize)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte mismatch at offset %d: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestWritePageWrongSize(t *testing.T) {
	dm, _ := newTestDiskManager(t)
	pageID := dm.AllocatePage()

	tooShort := make([]byte, PageSize-1)
	if err := dm.WritePage(pageID, tooShort); err == nil {
		t.Fatalf("expected error writing undersized page, got nil")
	}

	tooLong := make([]byte, PageSize+1)
	if err := dm.WritePage(pageID, tooLong); err == nil {
		t.Fatalf("expected error writing oversized page, got nil")
	}
}

func TestReadUnwrittenPage(t *testing.T) {
	dm, _ := newTestDiskManager(t)

	// Allocate a high page ID without ever writing pages before it.
	var pageID int32 = 100
	for dm.nextPageID <= pageID {
		dm.AllocatePage()
	}

	// This asserts your chosen behavior: reading a page that was never
	// written should return an error (design choice (a) from the writeup).
	// If you instead chose to return a zero-filled page, change this
	// test to check for that instead.
	_, err := dm.ReadPage(pageID)
	if err == nil {
		t.Fatalf("expected error reading unwritten page %d, got nil", pageID)
	}
}

func TestAllocatePageSequential(t *testing.T) {
	dm, _ := newTestDiskManager(t)

	first := dm.AllocatePage()
	second := dm.AllocatePage()
	third := dm.AllocatePage()

	if second != first+1 {
		t.Fatalf("expected sequential page IDs, got %d then %d", first, second)
	}
	if third != second+1 {
		t.Fatalf("expected sequential page IDs, got %d then %d", second, third)
	}
}

func TestPersistenceAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "persist.db")

	dm1, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("NewDiskManager failed: %v", err)
	}

	pageID := dm1.AllocatePage()
	want := make([]byte, PageSize)
	copy(want, []byte("persisted data"))

	if err := dm1.WritePage(pageID, want); err != nil {
		t.Fatalf("WritePage failed: %v", err)
	}
	if err := dm1.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Reopen as a fresh DiskManager instance backed by the same file.
	dm2, err := NewDiskManager(path)
	if err != nil {
		t.Fatalf("NewDiskManager (reopen) failed: %v", err)
	}
	defer dm2.Close()

	got, err := dm2.ReadPage(pageID)
	if err != nil {
		t.Fatalf("ReadPage after reopen failed: %v", err)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte mismatch at offset %d after reopen: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestFileActuallyCreatedOnDisk(t *testing.T) {
	dm, path := newTestDiskManager(t)
	_ = dm

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected db file to exist at %s: %v", path, err)
	}
}
