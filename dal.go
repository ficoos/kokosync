package main

import (
	"database/sql"
	_ "embed"
	"fmt"

	"github.com/ficoos/kokosync/kosync"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schema string

type DAL struct {
	db *sql.DB
}

/*
INSERT INTO document_hashs(book_id, document_hash)

		VALUES(?, ?)
	  	ON CONFLICT(book_id) DO UPDATE SET document_hash=excluded.document_hash
*/
func (dal *DAL) UpdateProgress(progress *kosync.Progress) error {
	_, err := dal.db.Exec(`
		INSERT INTO progress (
			document,
			progress,
			percentage,
			device_id,
			device)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(document) DO UPDATE SET 
			progress=excluded.progress,
			percentage=excluded.percentage,
			device_id=excluded.device_id,
			device=excluded.device
	`,
		progress.Document,
		progress.Progress,
		progress.Percentage,
		progress.DeviceID,
		progress.Device,
	)
	return err
}

func (dal *DAL) GetProgress(document string) (*kosync.Progress, error) {
	row := dal.db.QueryRow(`
	SELECT document, progress, percentage, device_id, device
	FROM progress
	WHERE document = ?
	`, document)
	var res kosync.Progress
	err := row.Scan(
		&res.Document,
		&res.Progress,
		&res.Percentage,
		&res.DeviceID,
		&res.Device)
	if err != nil {
		return nil, err
	}
	return &res, nil
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
