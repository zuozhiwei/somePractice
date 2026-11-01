package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"skilltree-go/internal/store"
	"testing"
)

func TestBackupAndNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.db")
	dest := filepath.Join(dir, "backup.db")
	db, err := store.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = store.Migrate(db); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO users(username,password,role,total_xp) VALUES('test','hash','EXTERNAL',7)`); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SKILLTREE_DB", src)
	t.Setenv("SKILLTREE_BACKUP_OUTPUT", dest)
	if err = run(); err != nil {
		t.Fatal(err)
	}
	bak, err := sql.Open("sqlite", readOnlyURI(dest))
	if err != nil {
		t.Fatal(err)
	}
	defer bak.Close()
	var xp int
	if err = bak.QueryRow(`SELECT total_xp FROM users WHERE username='test'`).Scan(&xp); err != nil || xp != 7 {
		t.Fatalf("backup xp=%d err=%v", xp, err)
	}
	before, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if err = run(); err == nil {
		t.Fatal("backup overwrote destination")
	}
	after, err := os.ReadFile(dest)
	if err != nil || string(before) != string(after) {
		t.Fatal("existing backup changed")
	}
}
