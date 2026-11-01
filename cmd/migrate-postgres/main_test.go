package main

import (
	"os"
	"path/filepath"
	"testing"

	"skilltree-go/internal/store"
)

func TestTypeMappingAndDigest(t *testing.T) {
	if sqliteType("jsonb") != "TEXT" || sqliteType("boolean") != "INTEGER" || sqliteType("uuid") != "UNKNOWN" {
		t.Fatal("unexpected source type mapping")
	}
	a, err := digest([]any{int64(1), "text", nil})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := digest([]any{int64(1), "text", nil})
	c, _ := digest([]any{int64(2), "text", nil})
	if a != b || a == c {
		t.Fatal("digest does not detect content changes")
	}
}

func TestExistingDestinationIsNeverOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.db")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SKILLTREE_SOURCE_DATABASE_URL", "postgres://unused")
	t.Setenv("SKILLTREE_MIGRATION_OUTPUT", path)
	if err := run(); err == nil {
		t.Fatal("expected destination conflict")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "keep" {
		t.Fatal("existing destination changed")
	}
}
func TestReadRowNormalizesJSON(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query(`SELECT '{"z":2,"a":1}', 1`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("missing row")
	}
	v, err := readRow(rows, []column{{typ: "jsonb"}, {typ: "boolean"}})
	if err != nil {
		t.Fatal(err)
	}
	if v[0] != `{"a":1,"z":2}` || v[1] != int64(1) {
		t.Fatalf("unexpected values %v", v)
	}
}
