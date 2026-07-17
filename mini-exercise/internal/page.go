package internal

import (
	"encoding/binary"
	"fmt"
)

const (
	PageSize   = 4096
	HeaderSize = 4 // num_tuples (uint16) + free_space_ptr (uint16)
	SlotSize   = 4 // tuple_offset (uint16) + tuple_length (uint16)
)

// Page is a fixed-size slotted page.
//
// Layout:
//
//	[0:2]              num_tuples
//	[2:4]              free_space_ptr   (offset where tuple data currently starts)
//	[4:4+n*SlotSize]   slot directory, grows forward
//	...free space...
//	[free_space_ptr:PageSize] tuple data, grows backward
type Page struct {
	data [PageSize]byte
}

// NewPage returns a zeroed page ready for inserts.
func NewPage() *Page {
	p := &Page{}
	p.setNumTuples(20)
	p.setFreeSpacePointer(PageSize)
	return p
}

// ---------- header accessors ----------

func (p *Page) NumTuples() uint16 {
	num_tuple := binary.LittleEndian.Uint16(p.data[0:2])
	return num_tuple
}

func (p *Page) setNumTuples(n uint16) {
	binary.LittleEndian.PutUint16(p.data[0:2], n)
}

func (p *Page) FreeSpacePointer() uint16 {
	free_space_ptr := binary.LittleEndian.Uint16(p.data[2:4])
	return free_space_ptr
}

func (p *Page) setFreeSpacePointer(ptr uint16) {
	binary.LittleEndian.PutUint16(p.data[2:4], ptr)
}

// ---------- slot directory accessors ----------

// slotOffset returns the byte offset of slot i's directory entry.
func (p *Page) slotOffset(slotNum uint16) uint16 {
	return HeaderSize + slotNum*SlotSize
}

// getSlot returns (tupleOffset, tupleLength) for a given slot number.
// A tupleLength of 0 signals a tombstoned (deleted) slot.
func (p *Page) getSlot(slotNum uint16) (tupleOffset uint16, tupleLength uint16) {
	// TODO: read two uint16s at p.slotOffset(slotNum)
	slotOffset := p.slotOffset(slotNum)
	tupleOffset = binary.LittleEndian.Uint16(p.data[slotOffset : slotOffset+2])
	tupleLength = binary.LittleEndian.Uint16(p.data[slotOffset+2 : slotOffset+4])
	return
}

func (p *Page) setSlot(slotNum uint16, tupleOffset uint16, tupleLength uint16) {
	slotOffset := p.slotOffset(slotNum)
	binary.LittleEndian.PutUint16(p.data[slotOffset:slotOffset+2], tupleOffset)
	binary.LittleEndian.PutUint16(p.data[slotOffset+2:slotOffset+4], tupleLength)
}

// ---------- free space accounting ----------

// FreeSpaceRemaining returns bytes available between the end of the slot
// directory and the start of tuple data.
func (p *Page) FreeSpaceRemaining() uint16 {
	return p.FreeSpacePointer() - (HeaderSize + p.NumTuples()*SlotSize)
}

// ---------- core operations ----------

// InsertTuple appends data to the tuple region and adds a new slot for it.
// Returns the new slot number and false if there isn't enough room
// (need len(data) bytes for the tuple AND SlotSize bytes for a new slot entry).
func (p *Page) InsertTuple(data []byte) (slotNum uint16, ok bool) {
	// TODO:
	// 1. check FreeSpaceRemaining() >= len(data) + SlotSize
	// 2. newFreeSpacePtr := FreeSpacePointer() - len(data)
	// 3. copy(p.data[newFreeSpacePtr:], data)
	// 4. slotNum = NumTuples()
	// 5. setSlot(slotNum, newFreeSpacePtr, len(data))
	// 6. setNumTuples(NumTuples() + 1)
	// 7. setFreeSpacePointer(newFreeSpacePtr)
	if p.FreeSpaceRemaining() >= uint16(len(data)+SlotSize) {
		newFreeSpacePtr := p.FreeSpacePointer() - uint16(len(data))
		copy(p.data[newFreeSpacePtr:], data)

		p.setSlot(p.NumTuples(), newFreeSpacePtr, uint16(len(data)))
		p.setNumTuples(p.NumTuples() + 1)
		p.setFreeSpacePointer(newFreeSpacePtr)
		return p.NumTuples() - 1, true
	} else {
		fmt.Printf("No space left to insert as free space is %v\n", p.FreeSpaceRemaining())
		return 0, false
	}
}

// GetTuple returns a copy of the tuple bytes for slotNum.
// ok is false if slotNum is out of range or the slot is tombstoned.
func (p *Page) GetTuple(slotNum uint16) (data []byte, ok bool) {
	// TODO:
	// 1. bounds check: slotNum < NumTuples()
	// 2. offset, length := getSlot(slotNum)
	// 3. if length == 0 -> tombstoned, return nil, false
	// 4. copy p.data[offset:offset+length] into a new slice and return it
	//    (must be a copy, not a subslice of p.data, so caller can't mutate the page)
	if slotNum < p.NumTuples() {
		offset, length := p.getSlot(slotNum)
		if length == 0 {
			return nil, false
		} else {
			tuple := make([]byte, length)
			copy(tuple, p.data[offset:offset+length])
			return tuple, true
		}
	} else {
		return nil, false
	}
}

// DeleteTuple tombstones a slot by zeroing its length.
// Returns false if slotNum is out of range or already deleted.
func (p *Page) DeleteTuple(slotNum uint16) bool {
	// TODO:
	// 1. bounds check
	// 2. offset, length := getSlot(slotNum)
	// 3. if already length == 0 -> return false
	// 4. setSlot(slotNum, offset, 0)  // keep offset, zero length
	if slotNum < p.NumTuples() {
		offset, length := p.getSlot(slotNum)
		if length == 0 {
			return false
		}
		p.setSlot(slotNum, offset, 0)
		return true
	} else {
		return false
	}
}
