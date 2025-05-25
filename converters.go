package main

import (
	"encoding/xml"
	"fmt"

	"github.com/ficoos/moonkosync/komga"
	"github.com/ficoos/moonkosync/kosync"
	"github.com/ficoos/moonkosync/xpointer"
)

func KOReaderProgressToMoonReaderProgress(krp *kosync.Progress, client *komga.Client, bookID string) (*MoonReaderProgress, error) {
	progress, err := xpointer.Parse(krp.Progress)
	if err != nil {
		return nil, fmt.Errorf("parse kosync progress: %s", err)
	}

	fragment := progress.Fragment()
	body, err := fetchPage(client, bookID, fragment)
	if err != nil {
		return nil, fmt.Errorf("get fragment: %s", err)
	}
	defer body.Close()
	dec := xml.NewDecoder(body)
	offset, err := progress.CountCharsToPosition(dec)
	if err != nil {
		return nil, fmt.Errorf("count chars: %s", err)
	}

	return &MoonReaderProgress{
		AddedToLibrary: 0, // TODO
		Fragment:       uint64(fragment),
		Unknown:        0,
		CharOffset:     offset,
		TotalProgress:  krp.Progress,
	}, nil
}
