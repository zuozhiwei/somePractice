package curriculum

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"skilltree-go/internal/store"
)

func TestEditionIsReachableAndRepeatable(t *testing.T) {
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(filepath.Join(t.TempDir(), "course.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = store.Migrate(db); err != nil {
		t.Fatal(err)
	}
	if _, err = p.Publish(db, true, false); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.QueryRow("SELECT count(*) FROM levels").Scan(&count); err != nil || count != 0 {
		t.Fatalf("rehearsal wrote %d levels: %v", count, err)
	}
	if _, err = p.Publish(db, true, true); err != nil {
		t.Fatal(err)
	}
	var chapters, levels, tasks int
	if err = db.QueryRow("SELECT (SELECT count(*) FROM maps),(SELECT count(*) FROM levels),(SELECT count(*) FROM tasks)").Scan(&chapters, &levels, &tasks); err != nil || chapters != 8 || levels != 46 || tasks != 179 {
		t.Fatal(chapters, levels, tasks, err)
	}
	reachable := map[int]bool{}
	for _, l := range p.Levels {
		for _, id := range l.Prereqs {
			if !reachable[id] {
				t.Fatalf("unreachable prerequisite %d for level %d", id, l.ID)
			}
		}
		reachable[l.ID] = true
		if l.guideBody == "" || l.guideTitle == "" {
			t.Fatal(l.ID)
		}
	}
	for _, item := range p.Tasks {
		var criteria []struct {
			Criteria string
			Weight   int
		}
		if err = json.Unmarshal(item.Criteria, &criteria); err != nil {
			t.Fatal(err)
		}
		total := 0
		for _, c := range criteria {
			if c.Criteria == "" {
				t.Fatal(item.ID)
			}
			total += c.Weight
		}
		if total != 100 || item.body == "" || !reachable[item.Level] {
			t.Fatalf("task %d: total=%d", item.ID, total)
		}
	}
	if _, err = db.Exec("UPDATE tasks SET description='Edited by teacher' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	msg, err := p.Publish(db, false, true)
	if err != nil || msg != "already published" {
		t.Fatal(msg, err)
	}
	var body string
	if err = db.QueryRow("SELECT description FROM tasks WHERE id=1").Scan(&body); err != nil || body != "Edited by teacher" {
		t.Fatal(body, err)
	}
}

func TestImportRollsBackOnLateContentConflict(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "edited.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = store.Migrate(db); err != nil {
		t.Fatal(err)
	}
	p, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Publish(db, true, true); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("DELETE FROM course_editions"); err != nil {
		t.Fatal(err)
	}
	// Mimic a partial import, where the new edition has an unexpected ID collision.
	ed := Catalog{Chapters: []chapter{{ID: 93, Name: "new chapter"}}, Levels: []level{{ID: 1, Name: "collision"}}}
	if _, err = ed.Publish(db, false, true); err == nil {
		t.Fatal("expected error")
	}
	var id int
	if err = db.QueryRow("SELECT id FROM maps WHERE id=93").Scan(&id); err != sql.ErrNoRows {
		t.Fatal("partial publication", err)
	}
}
