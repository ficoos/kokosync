package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ficoos/kokosync/kosync"
)

type BridgeImpl struct {
	upstream *url.URL
	dal      *DAL
}

func NewStore(upstream *url.URL, database string) (*BridgeImpl, error) {
	dal, err := NewDAL(database)
	if err != nil {
		return nil, fmt.Errorf("open database: %s", err)
	}

	return &BridgeImpl{upstream: upstream, dal: dal}, nil
}

// Authorize implements kosync.Server.
func (s *BridgeImpl) Authorize(auth *kosync.Auth) error {
	log.Printf("authorize: auth=%s", auth.Key)
	us := kosync.NewClient(s.upstream, auth.User, auth.Key)
	return us.Authorize()
}

// GetProgress implements kosync.Store.
func (s *BridgeImpl) GetProgress(auth *kosync.Auth, documentHash string) (*kosync.Progress, error) {
	log.Printf("get progress: auth=%s, document-hash=%s", auth, documentHash)
	us := kosync.NewClient(s.upstream, auth.User, auth.Key)
	err := us.Authorize()
	if err != nil {
		return nil, fmt.Errorf("authorize: %s", err)
	}

	p, err := s.dal.GetProgress(documentHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, kosync.ErrDocNotFound
		}
		return nil, fmt.Errorf("get progress from db [document=%s]: %s", documentHash, err)
	}

	return p, nil
}

// UpdateProgress implements kosync.Store.
func (s *BridgeImpl) UpdateProgress(auth *kosync.Auth, progress *kosync.Progress) (*kosync.UpdateProgressResult, error) {
	log.Printf("update progress: auth=%s, progress=%s", auth, progress)
	us := kosync.NewClient(s.upstream, auth.User, auth.Key)
	err := us.Authorize()
	if err != nil {
		return nil, fmt.Errorf("authorize: %s", err)
	}

	err = s.dal.UpdateProgress(progress)
	if err != nil {
		return nil, fmt.Errorf("save progres to db [document=%s]: %s", progress.Document, err)
	}
	_, err = us.UpdateProgress(progress)
	if err != nil {
		log.Printf("warning: update upstream: %s", err)
	}

	return &kosync.UpdateProgressResult{
		Document:  progress.Document,
		Timestamp: time.Now().Unix(),
	}, nil
}

var _ kosync.Server = &BridgeImpl{}

func main() {
	conf, err := ConfigFromEnvironment()
	if err != nil {
		log.Fatalf("load config from environemnt: %s", err)
	}

	srv, err := NewStore(conf.UpstreamURL, conf.DBPath)
	if err != nil {
		log.Fatalf("initialize server: %s", err)
	}

	l, err := net.Listen("tcp4", conf.ListenAddress)
	if err != nil {
		log.Fatalf("bind address: %s", err)
	}

	http.Serve(l, kosync.NewServer(srv))
}
