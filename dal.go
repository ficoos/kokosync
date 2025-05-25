package main

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"iter"
	"regexp"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schema string

type DAL struct {
	db *sql.DB
}

type BookCacheItem struct {
	Title   string
	Author  string
	ID      string
	FileURL string
}

var OmitChars = regexp.MustCompile("['`]")
var Replacements = map[string]string{
	"&": "and",
}

var convertToSpace = regexp.MustCompile(`[^\w\d]`)

func (b *BookCacheItem) NormalizedTitle() string {
	return NormalizeString(fmt.Sprintf("%s %s", b.Title, b.Author))
}

func NormalizeString(s string) string {
	s = OmitChars.ReplaceAllString(s, "")
	for from, to := range Replacements {
		s = strings.ReplaceAll(s, from, to)
	}
	s = convertToSpace.ReplaceAllString(s, " ")

	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}

func (dal *DAL) UpdateMRP(mrp *MoonReaderProgress) error {
	_, err := dal.db.Exec(
		`INSERT INTO mrp
		  (added_to_library, fragment, char_offset, total_progress)
		VALUES
		  (?, ?, ?, ?)`,
		mrp.AddedToLibrary,
		mrp.Fragment,
		mrp.CharOffset,
		mrp.TotalProgress)
	return err
}

func (dal *DAL) UpdateKRP(krp *KOReaderProgress) error {
	_, err := dal.db.Exec(
		`INSERT INTO mrp
		  (document_hash, fragment, selector, total_progress, timestamp, device_id)
		VALUES
		  (?, ?, ?, ?, ?, ?)`,
		krp.DocumentHash,
		krp.Fragment,
		krp.Selector,
		krp.TotalProgress,
		krp.Timestamp,
		krp.DeviceID)
	return err
}

func (dal *DAL) BookIDForMRP(addedToLibrary int64) (string, error) {
	// TODO

	return "", nil
}

func (dal *DAL) BookIDForKRP(documentHash string) (string, error) {
	// TODO

	return "", nil
}

func (dal *DAL) MatchBook(bookID, documentHash string, addedToLibrary int64) error {
	// TODO

	return nil
}

func (DAL *DAL) DocumentHash(bookID string) (string, error) {
	row := DAL.db.QueryRow("SELECT document_hash FROM document_hashs WHERE book_id = ?", bookID)
	var result string
	err := row.Scan(&result)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (DAL *DAL) UpdateDocumentHash(bookID, documentHash string) error {
	_, err := DAL.db.Exec(`
	INSERT INTO document_hashs(book_id, document_hash)
	VALUES(?, ?)
  	ON CONFLICT(book_id) DO UPDATE SET document_hash=excluded.document_hash;
	`, bookID, documentHash)

	return err
}

func (dal *DAL) FindBook(title string) (*BookCacheItem, error) {
	title = NormalizeString(title)
	res, err := dal.db.Query(`
	SELECT title, author, file_url, books.book_id
	FROM books
	INNER JOIN (
		SELECT book_id
		FROM books_fts
		WHERE normalized_title MATCH ?
	) AS fts ON books.book_id = fts.book_id`, title)

	if err != nil {
		return nil, err
	}

	defer res.Close()

	foundResult := false

	var book BookCacheItem
	for res.Next() {
		if foundResult {
			return nil, fmt.Errorf("no exact match found")
		}

		if err := res.Scan(&book.Title, &book.Author, &book.FileURL, &book.ID); err != nil {
			return nil, fmt.Errorf("reading row: %s", err)
		}
		foundResult = true
	}

	if err := res.Err(); err != nil {
		return nil, fmt.Errorf("fetching result: %s", err)
	}

	if !foundResult {
		return nil, fmt.Errorf("no match found")
	}

	return &book, nil
}

func (dal *DAL) FillBookCache(it iter.Seq2[BookCacheItem, error]) error {
	tx, err := dal.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %s", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}

		tx.Commit()
	}()
	// Clear the existing list
	_, err = tx.Exec("DELETE FROM books")
	if err != nil {
		return fmt.Errorf("clear books table: %s", err)
	}
	_, err = tx.Exec("DELETE FROM books_fts")
	if err != nil {
		return fmt.Errorf("clear fts table: %s", err)
	}

	var query strings.Builder
	params := make([]any, 0, 100)
	batchSize := 0
	batchCount := 0
	submitBatch := func() error {
		batchCount++
		_, err = tx.Exec(query.String(), params...)
		if err != nil {
			return fmt.Errorf("batch insert [%d] : %s", batchCount, err)
		}
		query.Reset()
		params = make([]any, 0, 100)

		batchSize = 0
		return nil
	}
	for item, err := range it {
		if err != nil {
			return fmt.Errorf("iterate entries: %s", err)
		}
		if batchSize == 100 {
			err = submitBatch()
			if err != nil {
				return err
			}
		}
		if batchSize == 0 {
			query.WriteString("INSERT OR IGNORE INTO books (title, author, book_id, file_url, normalized_title) VALUES (?, ?, ?, ?, ?)")
		} else {
			query.WriteString(",(?, ?, ?, ?, ?)")
		}

		batchSize++
		params = append(params, item.Title, item.Author, item.ID, item.FileURL, item.NormalizedTitle())
	}

	_, err = tx.Exec("INSERT INTO books_fts SELECT normalized_title, book_id FROM books")
	if err != nil {
		return fmt.Errorf("populate fts table: %s", err)
	}

	if batchSize > 0 {
		return submitBatch()
	}

	return nil
}

func NewDAL(path string) (*DAL, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open databse: %s", err)
	}

	dal := &DAL{db: db}
	_, err = dal.db.Exec(schema)
	if err != nil {
		return nil, fmt.Errorf("initialize databse: %s", err)
	}

	return dal, nil
}
