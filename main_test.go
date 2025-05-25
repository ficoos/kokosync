package main

import (
	"net/url"
	"testing"

	"github.com/ficoos/moonkosync/komga"
)

const KEY string = "da54aed06adf41829d3dfbb817d38239"

func TestMoonProgressToKoreaderProgress(t *testing.T) {
	bookID := "0K623K58D9Z8B"
	mrp, err := Parse("1725870547829*40@0#1766:39.0%")

	if err != nil {
		t.Fatal("parse moon progress: ", err)
	}

	apiRoot, err := url.Parse("https://comics.mizrahi.cc")
	if err != nil {
		t.Fatal("parse api root: ", err)
	}
	client := komga.New(*apiRoot, KEY)
	krp, err := MoonReaderProgressToKOReaderProgress(mrp, client, bookID)
	if err != nil {
		t.Fatal("convert progress mrp -> krp: ", err)
	}

	t.Logf("mainfest: %s", krp.Selector)
	// mrp2, err := KOReaderProgressToMoonReaderProgress(krp, client, bookID)
	// if err != nil {
	// 	t.Fatal("convert progress: krp -> mrp", err)
	// }
	//t.Logf("mainfest: %d", mrp2.CharOffset)
}
