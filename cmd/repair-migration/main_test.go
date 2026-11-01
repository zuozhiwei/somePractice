package main

import (
	"path/filepath"
	"skilltree-go/internal/store"
	"testing"
)

func TestArchiveOrphansAndEnforceRelations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "copy.db")
	db, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Prepare(db); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO conversations(id,type) VALUES(1,'DIRECT')`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO conversation_members(id,conversation_id,user_id) VALUES(7,1,999)`); err != nil {
		t.Fatal(err)
	}
	db.Close()
	t.Setenv("SKILLTREE_MIGRATION_OUTPUT", path)
	if err = run(); err != nil {
		t.Fatal(err)
	}
	if err = run(); err != nil {
		t.Fatal(err)
	}
	db, err = store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var archived, live, version int
	if err = db.QueryRow(`SELECT count(*) FROM migration_orphan_members WHERE id=7 AND user_id=999`).Scan(&archived); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM conversation_members WHERE id=7`).Scan(&live); err != nil {
		t.Fatal(err)
	}
	if err = db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if archived != 1 || live != 0 || version != 3 {
		t.Fatalf("archived=%d live=%d version=%d", archived, live, version)
	}
	if _, err = db.Exec(`INSERT INTO conversation_members(conversation_id,user_id) VALUES(1,999)`); err == nil {
		t.Fatal("orphan accepted after repair")
	}
}
