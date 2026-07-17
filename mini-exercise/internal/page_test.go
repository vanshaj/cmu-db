package internal

import "testing"

func TestInsertAndGet(t *testing.T) {
	p := NewPage()

	slot, ok := p.InsertTuple([]byte("hello"))
	if !ok {
		t.Fatalf("insert failed unexpectedly")
	}

	got, ok := p.GetTuple(slot)
	if !ok {
		t.Fatalf("get failed unexpectedly")
	}
	if string(got) != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}
}

func TestInsertUntilFull(t *testing.T) {
	p := NewPage()
	tuple := make([]byte, 100)

	count := 0
	for {
		_, ok := p.InsertTuple(tuple)
		if !ok {
			break
		}
		count++
		if count > PageSize { // safety valve against infinite loop bugs
			t.Fatalf("insert never returned false — free space accounting is broken")
		}
	}
	if count == 0 {
		t.Fatalf("expected at least one successful insert")
	}
}

func TestDeleteTombstones(t *testing.T) {
	p := NewPage()
	slot, _ := p.InsertTuple([]byte("bye"))

	if ok := p.DeleteTuple(slot); !ok {
		t.Fatalf("delete failed unexpectedly")
	}

	if _, ok := p.GetTuple(slot); ok {
		t.Fatalf("expected GetTuple on deleted slot to return ok=false")
	}

	if ok := p.DeleteTuple(slot); ok {
		t.Fatalf("double delete should return false")
	}
}

func TestSlotStabilityAfterDelete(t *testing.T) {
	p := NewPage()
	s1, _ := p.InsertTuple([]byte("first"))
	s2, _ := p.InsertTuple([]byte("second"))

	p.DeleteTuple(s1)

	got, ok := p.GetTuple(s2)
	if !ok || string(got) != "second" {
		t.Fatalf("deleting slot %d corrupted slot %d: got=%q ok=%v", s1, s2, got, ok)
	}
}
