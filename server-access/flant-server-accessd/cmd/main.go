package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/flant/server-access/flant-server-accessd/db/sqlite"
	"github.com/flant/server-access/flant-server-accessd/system"
	"github.com/flant/server-access/flant-server-accessd/types"
)

type UserDatabase interface {
	Migrate() error
	Sync(ctx context.Context, uwg types.UsersWithGroups) error
	GetChanges(ctx context.Context, uwg types.UsersWithGroups) ([]types.User, []types.User, error)
}

var db UserDatabase

func syncHandler(w http.ResponseWriter, r *http.Request) {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Print(err)
		}
	}(r.Body)

	ctx, close := context.WithTimeout(context.Background(), time.Minute)
	defer close()

	switch r.Method {
	case "POST":
		var uwg types.UsersWithGroups
		d := json.NewDecoder(r.Body)
		err := d.Decode(&uwg)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		err = applyChanges(ctx, uwg)
		if err != nil {
			log.Fatal(err)
		}

		err = db.Sync(ctx, uwg)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "I can't do that.")
	}
}

func main() {
	database, err := sqlite.NewUserDatabase("/home/zuzzas/sqlite_tests/server_access.db")
	if err != nil {
		log.Fatal(err)
	}

	err = database.Migrate()
	if err != nil {
		log.Fatal(err)
	}

	db = database

	http.HandleFunc("/v1/sync", syncHandler)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func applyChanges(ctx context.Context, uwg types.UsersWithGroups) error {
	newUsers, oldUsers, err := db.GetChanges(ctx, uwg)
	if err != nil {
		return err
	}

	var sysOp system.Interface
	sysOp = system.NewSystemOperator()

	for _, oldUser := range oldUsers {
		err := sysOp.DeleteHomeDir(oldUser.HomeDir)
		if err != nil {
			return err
		}

		err = sysOp.BootUser(oldUser.Name)
		if err != nil {
			return err
		}
	}

	for _, newUser := range newUsers {
		err := sysOp.CreateHomeDir(newUser.HomeDir, int(newUser.Uid), int(newUser.Gid))
		if err != nil {
			return err
		}
	}

	return nil
}
