package main

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ficoos/moonkosync/komga"
	"github.com/ficoos/moonkosync/kosync"
	"github.com/ficoos/moonkosync/urlutil"
	"github.com/ficoos/moonkosync/xpointer"
	"golang.org/x/net/webdav"
)

type KOReaderProgress struct {
	DocumentHash  string
	Fragment      int
	Selector      string
	TotalProgress float64
	Timestamp     int64
	DeviceID      string
}

type MoonReaderProgress struct {
	AddedToLibrary uint64
	Fragment       uint64
	Unknown        uint64
	CharOffset     uint64
	TotalProgress  string
}

var MoonProgressFormat = regexp.MustCompile(`^(\d+)\*(\d+)@(\d+)#(\d+):([\d.]+)%$`)

func Parse(s string) (*MoonReaderProgress, error) {
	//1725870547829*40@0#1766:39.0%
	var err error

	match := MoonProgressFormat.FindStringSubmatch(s)
	if len(match) == 0 {
		return nil, fmt.Errorf("invalid moon reader progress: %s", s)
	}

	result := &MoonReaderProgress{}

	for i, fieldDef := range []struct {
		fieldPtr  *uint64
		fieldName string
	}{
		{&result.AddedToLibrary, "AddedToLibrary"},
		{&result.Fragment, "Fragment"},
		{&result.Unknown, "Unknown"},
		{&result.CharOffset, "CharOffset"},
	} {
		*fieldDef.fieldPtr, err = strconv.ParseUint(match[i+1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %s", fieldDef.fieldName, err)
		}
	}

	result.TotalProgress = match[5]
	if err != nil {
		return nil, fmt.Errorf("parse TotalProgress: %s", err)
	}

	return result, nil
}

func (p *MoonReaderProgress) String() string {
	return fmt.Sprintf("%d*%d@%d#%d:%s", p.AddedToLibrary, p.Fragment, p.Unknown, p.CharOffset, p.TotalProgress)
}

type SyncFs struct {
	komga  *komga.Client
	kosync *kosync.Client
	dal    *DAL
	fs     webdav.FileSystem
}

func discard(r io.Reader, n int64) (int64, error) {
	buff := make([]byte, 1024)

	tr := int64(0)
	for tr < n {
		b := buff
		remaining := n - tr
		if remaining < int64(len(buff)) {
			b = buff[:remaining]
		}
		r, err := r.Read(b)
		tr += int64(r)
		if err != nil {
			return tr, err
		}
	}

	return tr, nil
}

func calculateDocumentHash(komga *komga.Client, u *url.URL) (string, error) {
	var offsets = []int64{
		0,
		1024,
		1024 << 2,
		1024 << (2 * 2),
		1024 << (2 * 3),
		1024 << (2 * 4),
		1024 << (2 * 5),
		1024 << (2 * 6),
		1024 << (2 * 7),
		1024 << (2 * 8),
		1024 << (2 * 9),
		1024 << (2 * 10),
	}

	h := md5.New()
	doc, err := komga.Get(u)
	if err != nil {
		return "", fmt.Errorf("get document: %s", err)
	}

	defer doc.Close()

	buff := make([]byte, 1024)

	offset := int64(0)
	for _, nextOffset := range offsets {
		r, err := discard(doc, nextOffset-offset)
		offset += r
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("reading document %s: %s", u, err)
		}

		n, err := doc.Read(buff)
		offset += int64(n)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return "", fmt.Errorf("reading document %s: %s", u, err)
		}
		h.Write(buff[:n])
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

var _ webdav.FileSystem = &SyncFs{}

// Mkdir implements webdav.FileSystem.
func (s *SyncFs) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return nil
}

// OpenFile implements webdav.FileSystem.
func (s *SyncFs) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	parts := strings.Split(name, "/")
	// apiKey := parts[0] TODO
	fname := parts[len(parts)-1]
	ext := filepath.Ext(fname)
	if strings.ToLower(ext) != ".po" {
		// TODO: this could be a problem
		return s.fs.OpenFile(ctx, name, flag, perm)
	}
	fname = strings.TrimSuffix(fname, ext)
	ext = filepath.Ext(fname) // This should be something like epub, cbz
	title := strings.TrimSuffix(fname, ext)
	book, err := s.dal.FindBook(title)
	if err != nil {
		return nil, fmt.Errorf("find book: %s", err)
	}

	dhash, err := s.dal.DocumentHash(book.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get document hash from cache")
	}

	if dhash == "" {
		u, err := url.Parse(book.FileURL)
		if err != nil {
			return nil, fmt.Errorf("parse book [%s] file url: %s", book.ID, err)
		}
		dhash, err = calculateDocumentHash(s.komga, u)
		if err != nil {
			return nil, fmt.Errorf("calculate document hash for book [%s]: %s", book.ID, err)
		}

		err = s.dal.UpdateDocumentHash(book.ID, dhash)
		if err != nil {
			return nil, fmt.Errorf("update document hash for book [%s]: %s", book.ID, err)
		}
	}

	f, err := NewProgressFile(book.ID, dhash, s.kosync, s.komga)
	if err != nil {
		return nil, fmt.Errorf("initialize progress file: %s", err)
	}

	log.Printf("OpenFile %s", name)
	return f, nil
}

// RemoveAll implements webdav.FileSystem.
func (s *SyncFs) RemoveAll(ctx context.Context, name string) error {
	// No op
	return nil
}

// Rename implements webdav.FileSystem.
func (s *SyncFs) Rename(ctx context.Context, oldName string, newName string) error {
	return webdav.ErrNotImplemented
}

// Stat implements webdav.FileSystem.
func (s *SyncFs) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	return nil, webdav.ErrNotImplemented
}

func findSelectorAtOffset(r io.Reader, offset uint64) (string, error) {
	dec := xml.NewDecoder(r)
	selector := xpointer.XPointerBuilder{}
	for {
		token, _ := dec.Token()
		if token == nil {
			return "", fmt.Errorf("unexpected EOF")
		}
		switch elem := token.(type) {
		case xml.EndElement:
			selector.Pop()
		case xml.Comment:
		case xml.ProcInst:
		case xml.Directive:
			//Skip
			break
		case xml.StartElement:
			selector.Push(elem.Name.Local)
		case xml.CharData:
			runeCount := utf8.RuneCount(elem)
			if uint64(runeCount) < offset {
				offset -= uint64(runeCount)
				break
			}

			return fmt.Sprintf("%s/text().%d", selector.String(), offset), nil
		}
	}
}

func fetchPage(client *komga.Client, bookID string, fragment int) (io.ReadCloser, error) {
	manifest, err := client.WebPubManifest(bookID)
	if err != nil {
		return nil, fmt.Errorf("fetch webpub manifset: %s", err)
	}

	if len(manifest.ReadingOrder) < fragment {
		return nil, fmt.Errorf("MRP fragment not in book range: mrp fragment = %d, book fragment count = %d", fragment, len(manifest.ReadingOrder))
	}

	frag := manifest.ReadingOrder[fragment]
	if frag.Type != "application/xhtml+xml" {
		return nil, fmt.Errorf("unsupported fragment type: %s", frag.Type)
	}

	u, err := url.Parse(frag.Href)
	if err != nil {
		return nil, fmt.Errorf("unsupported fragment href: %s", err)
	}

	body, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("get fragment: %s", err)
	}

	return body, nil
}

func MoonReaderProgressToKOReaderProgress(mrp *MoonReaderProgress, client *komga.Client, bookID string) (*KOReaderProgress, error) {
	body, err := fetchPage(client, bookID, int(mrp.Fragment))
	if err != nil {
		return nil, fmt.Errorf("get fragment: %s", err)
	}

	defer body.Close()

	selector, err := findSelectorAtOffset(body, mrp.CharOffset)
	if err != nil {
		return nil, fmt.Errorf("find selector: %s", err)
	}

	return &KOReaderProgress{
		Fragment: int(mrp.Fragment),
		Selector: selector,
	}, nil
}

func scrape(client *komga.Client, dal *DAL) error {
	return dal.FillBookCache(func(yield func(item BookCacheItem, err error) bool) {
		for entry, err := range client.IterateOPDSFeed("libraries") {
			if err != nil {
				yield(BookCacheItem{}, err)
				return
			}
			for _, author := range entry.Authors {
				var fileUrl string
				for _, link := range entry.Links {
					if link.Rel == "http://opds-spec.org/acquisition" {
						fileUrl = link.Href
					}
				}
				ok := yield(BookCacheItem{
					Title:   entry.Title,
					Author:  author.Name,
					FileURL: fileUrl,
					ID:      entry.ID,
				}, nil)
				if !ok {
					return
				}
			}
		}
	})
}

type Store struct {
}

// Authorize implements kosync.Server.
func (s *Store) Authorize(auth *kosync.Auth) error {
	log.Printf("authorize: auth=%s", auth)

	return nil
}

// GetProgress implements kosync.Store.
func (s *Store) GetProgress(auth *kosync.Auth, documentHash string) (*kosync.Progress, error) {
	log.Printf("get progress: auth=%s, document-hash=%s", auth, documentHash)
	return nil, kosync.ErrDocNotFound
}

// UpdateProgress implements kosync.Store.
func (s *Store) UpdateProgress(auth *kosync.Auth, progress *kosync.Progress) (*kosync.UpdateProgressResult, error) {
	log.Printf("update progress: auth=%s, progress=%s", auth, progress)
	path, err := xpointer.Parse(progress.Progress)
	if err != nil {

	}
	return nil, kosync.ErrBadRequest
}

var _ kosync.Server = &Store{}

func main() {
	conf, err := ConfigFromEnvironment()
	if err != nil {
		log.Fatalf("load config from environemnt: %s", err)
	}

	komgaClient := komga.New(*conf.KomgaAPIRoot, conf.KomgaAPIKey)
	kosyncClient := kosync.NewClient(urlutil.Join(conf.KomgaAPIRoot, "/koreader"), conf.KomgaAPIKey, "ingored")

	dal, err := NewDAL(conf.DBPath)
	if err != nil {
		log.Fatalf("initialized DAL: %s", err)
	}
	// log.Print("Scarping books...")
	// err = scrape(komgaClient, dal)
	// log.Print("Finished scarping books...")
	if err != nil {
		log.Fatalf("scrape books: %s", err)
	}
	davServer := &webdav.Handler{
		Prefix: "/",
		FileSystem: &SyncFs{
			komga:  komgaClient,
			dal:    dal,
			fs:     webdav.NewMemFS(),
			kosync: kosyncClient,
		},
		LockSystem: webdav.NewMemLS(),
		Logger:     nil,
	}

	l, err := net.Listen("tcp4", conf.ListenAddress)
	if err != nil {
		log.Fatalf("bind address: %s", err)
	}

	kosyncServer := kosync.NewServer(&Store{})
	log.Printf("Started serving at: %s", conf.ListenAddress)

	root := http.NewServeMux()
	root.Handle("/dev/", http.StripPrefix("/dev", davServer))
	root.Handle("/kosync/", http.StripPrefix("/kosync", kosyncServer))

	http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request: [%s] %s", r.Method, r.RequestURI)
		root.ServeHTTP(w, r)
	}))
}
