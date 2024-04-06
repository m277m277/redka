package main

import (
	"database/sql"
	"log"
	"log/slog"

	"github.com/nalgeon/redka"
	driver "modernc.org/sqlite"
)

func main() {
	// An example of using Redka with modernc.org/sqlite driver.

	// modernc.org/sqlite uses a different driver name ("sqlite"), while
	// Redka expects "sqlite3". So we have to re-register it as "sqlite3".
	sql.Register("sqlite3", &driver.Driver{})

	db, err := redka.Open("data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Str().Set("name", "alice")
	slog.Info("set", "err", err)

	count, err := db.Key().Exists("name", "age", "city")
	slog.Info("exists", "count", count, "err", err)
}