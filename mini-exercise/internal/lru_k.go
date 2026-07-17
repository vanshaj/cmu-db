package internal

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

type FrameEntry struct {
	History   *list.List
	Evictable bool
}

type LRUKReplacer struct {
	mu     sync.Mutex
	Frames map[int]*FrameEntry
	K      int // this will be used to measure distance
}

func NewLRUKReplacer(k int) *LRUKReplacer {
	return &LRUKReplacer{
		Frames: make(map[int]*FrameEntry),
		K:      k,
	}
}

func (r *LRUKReplacer) RecordAccess(frameID int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v, ok := r.Frames[frameID]; ok {
		if v.History.Len() == r.K {
			v.History.Remove(v.History.Front())
		}
		v.History.PushBack(time.Now())
	} else {
		historyList := list.New()
		historyList.PushBack(time.Now())
		fEntry := &FrameEntry{
			History:   historyList,
			Evictable: false,
		}

		r.Frames[frameID] = fEntry
	}
}

func (r *LRUKReplacer) SetEvictable(frameID int, evictable bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v, ok := r.Frames[frameID]; ok {
		v.Evictable = evictable
		return nil
	} else {
		return fmt.Errorf("No such frame exists")
	}
}

func (r *LRUKReplacer) Evict() (frameID int, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	evictableFramesInfList := make([]int, 0)
	evictableFramesNonInfList := make([]int, 0)
	for k, v := range r.Frames {
		if v.Evictable {
			if v.History.Len() < r.K {
				evictableFramesInfList = append(evictableFramesInfList, k)
			} else {
				evictableFramesNonInfList = append(evictableFramesNonInfList, k)
			}
		}
	}

	var evictableFrameID int
	if len(evictableFramesInfList) > 0 {
		for index, v_fId := range evictableFramesInfList {
			if index == 0 {
				evictableFrameID = v_fId
			} else {
				currentFLatestAccess := r.Frames[v_fId].History.Back().Value.(time.Time)
				if currentFLatestAccess.Before(r.Frames[evictableFrameID].History.Back().Value.(time.Time)) {
					evictableFrameID = v_fId
				}
			}
		}
		delete(r.Frames, evictableFrameID)
		return evictableFrameID, true
	}
	if len(evictableFramesNonInfList) > 0 {
		for index, v_fId := range evictableFramesNonInfList {
			if index == 0 {
				evictableFrameID = v_fId
			} else {
				currentFLatestAccess := r.Frames[v_fId].History.Back().Value.(time.Time)
				if currentFLatestAccess.Before(r.Frames[evictableFrameID].History.Back().Value.(time.Time)) {
					evictableFrameID = v_fId
				}
			}
		}
		delete(r.Frames, evictableFrameID)
		return evictableFrameID, true
	}
	return evictableFrameID, false
}

func (r *LRUKReplacer) Remove(frameID int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Frames, frameID)
}

func (r *LRUKReplacer) Size() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	var countSize int
	for _, v := range r.Frames {
		if v.Evictable {
			countSize++
		}
	}
	return countSize
}
