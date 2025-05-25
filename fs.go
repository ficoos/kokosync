package main

import (
	"fmt"
	"io/fs"

	"github.com/ficoos/moonkosync/komga"
	"github.com/ficoos/moonkosync/kosync"
	"golang.org/x/net/webdav"
)

type ProgressFile struct {
	bookID       string
	documentHash string
	kosync       *kosync.Client
	mrp          *MoonReaderProgress
	offset       int64
}

func NewProgressFile(bookID, documentHash string, kosync *kosync.Client, komga *komga.Client) (*ProgressFile, error) {
	progress, err := kosync.Progress(documentHash)
	if err != nil {
		return nil, fmt.Errorf("fetch progress: %s", err)
	}

	mrp, err := KOReaderProgressToMoonReaderProgress(progress, komga, bookID)
	if err != nil {
		return nil, fmt.Errorf("convert to mrp: %s", err)
	}

	return &ProgressFile{
		bookID:       bookID,
		documentHash: documentHash,
		kosync:       kosync,
		mrp:          mrp,
	}, nil
}

// Close implements webdav.File.
func (p *ProgressFile) Close() error {
	// no op
	return nil
}

// Read implements webdav.File.
func (p *ProgressFile) Read(buff []byte) (int, error) {
	buf := []byte(p.mrp.String())
	n := copy(buf[p.offset:], buf)

	return n, nil
}

// Readdir implements webdav.File.
func (p *ProgressFile) Readdir(count int) ([]fs.FileInfo, error) {
	return nil, webdav.ErrNotImplemented
}

// Seek implements webdav.File.
func (p *ProgressFile) Seek(offset int64, whence int) (int64, error) {
	return -1, webdav.ErrNotImplemented
}

// Stat implements webdav.File.
func (p *ProgressFile) Stat() (fs.FileInfo, error) {
	return nil, webdav.ErrNotImplemented
}

// Write implements webdav.File.
func (p *ProgressFile) Write(buff []byte) (n int, err error) {
	mrp, err := Parse(string(buff))
	if err != nil {
		// TODO: log
		return 0, webdav.ErrForbidden
	}

	p.mrp = mrp

	return len(buff), nil
}

var _ webdav.File = &ProgressFile{}
