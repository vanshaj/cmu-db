package internal

import "fmt"

type DiskRequest struct {
	IsWrite    bool
	PageID     int32
	Data       []byte     // caller-allocated, PageSize bytes; read fills it, write reads from it
	ResultChan chan error //scheduler sends nil (success) or an error, then caller reads it
}

type DiskScheduler struct {
	diskManager *DiskManager
	requestChan chan DiskRequest
	stopChan    chan struct{}
}

func NewDiskScheduler(dm *DiskManager) *DiskScheduler {
	s := &DiskScheduler{
		diskManager: dm,
		requestChan: make(chan DiskRequest, 20),
		stopChan:    make(chan struct{}),
	}
	go s.startWorker()
	return s
}

func (s *DiskScheduler) handleRequest(req DiskRequest) {
	if req.IsWrite {
		err := s.diskManager.WritePage(req.PageID, req.Data)
		if err != nil {
			fmt.Println("Error while writing page to the disk manager")
		}
		req.ResultChan <- err
		return
	}
	data, err := s.diskManager.ReadPage(req.PageID)
	if err != nil {
		fmt.Println("Error while reading page from the disk manager")
		req.ResultChan <- err
		return
	}
	copy(req.Data, data)
	req.ResultChan <- nil

}

func (s *DiskScheduler) startWorker() {
loop:
	for {
		select {
		case req, ok := <-s.requestChan:
			if !ok {
				fmt.Println("Request Channel closed")
				break loop
			}
			s.handleRequest(req)
		case <-s.stopChan:
			fmt.Println("Quit Signal Received")
			pending := len(s.requestChan)
			for i := 0; i < pending; i++ {
				req := <-s.requestChan
				s.handleRequest(req)
			}
			break loop
		}
	}
}

func (s *DiskScheduler) Schedule(req DiskRequest) {
	s.requestChan <- req
}

func (s *DiskScheduler) Stop() {
	s.stopChan <- struct{}{}
}
