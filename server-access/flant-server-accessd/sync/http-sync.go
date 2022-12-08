package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/flant/negentropy/server-access/flant-server-accessd/db"
	"github.com/flant/negentropy/server-access/flant-server-accessd/types"
)

type Server struct {
	Address string
	URI     string
	DB      db.UserDatabase
}

func NewServer(addr string, uri string, dbInstance db.UserDatabase) *Server {
	return &Server{
		Address: addr,
		URI:     uri,
		DB:      dbInstance,
	}
}

func (s *Server) Start() error {
	http.HandleFunc(s.URI, s.syncHandler)
	return http.ListenAndServe(s.Address, nil)
}

func (s *Server) syncHandler(w http.ResponseWriter, r *http.Request) {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Print(err)
		}
	}(r.Body)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	switch r.Method {
	case "POST":
		var uwg types.UsersWithGroups
		d := json.NewDecoder(r.Body)
		err := d.Decode(&uwg)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		err = ApplyChanges(ctx, s.DB, uwg)
		if err != nil {
			println(err.Error())
			log.Fatal(err)
		}

		err = s.DB.Sync(ctx, uwg)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = fmt.Fprintf(w, "I can't do that.")
	}
}
