package kosync

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

var ErrDocNotFound = errors.New("document not found")
var ErrBadRequest = errors.New("bad request")
var ErrUnauthorized = errors.New("forbidden")

type Auth struct {
	User string
	Key  string
}

func translateError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, ErrDocNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrBadRequest) {
		return http.StatusBadRequest
	}
	if errors.Is(err, ErrUnauthorized) {
		return http.StatusForbidden
	}

	return http.StatusInternalServerError
}

func (a *Auth) String() string {
	return fmt.Sprintf("[user=%s key=REDACTED]", a.User)
}

func (a *Auth) Repr() string {
	return fmt.Sprintf("[user=%s key=%s]", a.User, a.Key)
}

type Server interface {
	UpdateProgress(auth *Auth, progress *Progress) (*UpdateProgressResult, error)
	GetProgress(auth *Auth, documentHash string) (*Progress, error)
	Authorize(auth *Auth) error
}

func extractAuth(r *http.Request) *Auth {
	var res Auth
	res.User = r.Header.Get("X-Auth-User")
	res.Key = r.Header.Get("X-Auth-Key")

	return &res
}

func NewServer(server Server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/auth", func(w http.ResponseWriter, r *http.Request) {
		auth := extractAuth(r)
		err := server.Authorize(auth)
		status := translateError(err)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(status)
		if status == http.StatusInternalServerError {
			log.Printf("ERROR: %s: %s", r.RequestURI, err)
		}
		if err == nil {
			w.Write([]byte(`{ "authorized": "OK" }`))
		}
	})
	mux.HandleFunc("GET /syncs/progress/{documenthash}", func(w http.ResponseWriter, r *http.Request) {
		hash := r.PathValue("documenthash")
		progress, err := server.GetProgress(extractAuth(r), hash)
		status := translateError(err)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(status)
		if status == http.StatusInternalServerError {
			log.Printf("ERROR: %s: %s", r.RequestURI, err)
		}
		if err != nil {
			return
		}

		enc := json.NewEncoder(w)
		err = enc.Encode(progress)
		if err != nil {
			log.Printf("ERROR: %s: %s", r.RequestURI, err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("PUT /syncs/progress", func(w http.ResponseWriter, r *http.Request) {
		dec := json.NewDecoder(r.Body)
		var p Progress
		err := dec.Decode(&p)
		if err != nil {
			log.Printf("ERROR: %s: %s", r.RequestURI, err)
			w.WriteHeader(http.StatusBadRequest)
		}
		res, err := server.UpdateProgress(extractAuth(r), &p)
		status := translateError(err)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(status)
		if status == http.StatusInternalServerError {
			log.Printf("ERROR: %s: %s", r.RequestURI, err)
		}
		if err != nil {
			return
		}

		enc := json.NewEncoder(w)
		err = enc.Encode(res)
		if err != nil {
			log.Printf("ERROR: %s: %s", r.RequestURI, err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	return mux
}
