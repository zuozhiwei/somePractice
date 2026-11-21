package store

import (
	"path/filepath"
	"testing"
)

func TestMentorshipRankMigrationPreservesTies(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "mentorship.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Prepare(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO users(id,username,password,total_xp,level) VALUES
			(1,'student','test-password',0,5),(2,'lower','test-password',0,4),(3,'equal','test-password',0,5);
		INSERT INTO mentorships(id,mentor_id,apprentice_id,status) VALUES
			(1,2,1,'ACTIVE'),(2,3,1,'ACTIVE');
	`); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[int]string{1: "ENDED", 2: "ACTIVE"} {
		var got string
		if err := db.QueryRow("SELECT status FROM mentorships WHERE id=?", id).Scan(&got); err != nil || got != want {
			t.Fatalf("relationship %d status = %q (%v), want %q", id, got, err, want)
		}
	}
}
