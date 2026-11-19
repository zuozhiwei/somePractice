// Command curriculum validates and publishes a course edition to a SQLite installation.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"skilltree-go/internal/curriculum"
	"skilltree-go/internal/store"
)

func main() {
	path := flag.String("db", "", "SQLite database path")
	apply := flag.Bool("apply", false, "commit the edition (default is a rehearsal)")
	initialize := flag.Bool("init", false, "install into an empty course database")
	flag.Parse()
	if *path == "" {
		log.Fatal("-db is required")
	}
	if _, err := os.Stat(*path); err != nil && !(*initialize && *apply && os.IsNotExist(err)) {
		log.Fatal(err)
	}
	pack, err := curriculum.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := store.Open(*path)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// Only bootstrap schema for an entirely empty database. Existing installations
	// are validated by Publish before any course change, including mistaken -init.
	var tables int
	if err = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tables); err != nil {
		log.Fatal(err)
	}
	if *initialize && *apply && tables == 0 {
		if err := store.Migrate(db); err != nil {
			log.Fatal(err)
		}
	}
	result, err := pack.Publish(db, *initialize, *apply)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: %s (%d chapters, %d levels, %d tasks)\n", curriculum.Edition, result, len(pack.Chapters), len(pack.Levels), len(pack.Tasks))
}
