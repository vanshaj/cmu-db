package internal

import (
	"fmt"
	"os"
	"sync"
)

type DiskManager struct {
	sync.RWMutex
	file       *os.File
	nextPageID int32
}

func NewDiskManager(filename string) (*DiskManager, error) {
	fd, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return &DiskManager{
		file:       fd,
		nextPageID: 0,
	}, nil
}

func (d *DiskManager) ReadPage(pageID int32) ([]byte, error) {
	offset := int64(pageID * PageSize)
	buf := make([]byte, PageSize)
	d.RLock()
	n, err := d.file.ReadAt(buf, offset)
	d.RUnlock()
	if err != nil {
		return nil, err
	}
	if n < PageSize {
		fmt.Printf("Read only %d bytes rather than complete %d bytes\n", n, PageSize)
	}
	return buf, nil
}

func (d *DiskManager) WritePage(pageId int32, data []byte) error {
	offset := int64(pageId * PageSize)
	if len(data) > PageSize {
		return fmt.Errorf("Too Long data to be written to page")
	}
	if len(data) < PageSize {
		return fmt.Errorf("Too Short data to be written to page")
	}
	d.Lock()
	n, err := d.file.WriteAt(data, offset)
	d.Unlock()
	if err != nil {
		return err
	}
	if n < len(data) {
		fmt.Printf("Read only %d bytes rather than complete %d bytes\n", n, len(data))
	}
	return nil
}

func (d *DiskManager) AllocatePage() int32 {
	currentPageId := d.nextPageID
	d.nextPageID++
	return currentPageId
}

func (d *DiskManager) Close() error {
	return d.file.Close()
}
