package curriculum

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
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
	var firstTasks string
	if err = db.QueryRow("SELECT group_concat(id) FROM (SELECT id FROM tasks WHERE level_id=1 ORDER BY sort_order,id)").Scan(&firstTasks); err != nil || firstTasks != "2,165,166,1,3,5,167,4" {
		t.Fatalf("unexpected beginner task order: %q %v", firstTasks, err)
	}
	reachable := map[int]bool{}
	lastTier := 0
	for _, l := range p.Levels {
		for _, id := range l.Prereqs {
			if !reachable[id] {
				t.Fatalf("unreachable prerequisite %d for level %d", id, l.ID)
			}
		}
		reachable[l.ID] = true
		if l.Tier < lastTier || l.Tier > lastTier+1 {
			t.Fatalf("difficulty jumps between %d and %d at level %d", lastTier, l.Tier, l.ID)
		}
		lastTier = l.Tier
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

func TestUpgradeKeepsProgressAndProtectsTeacherEdits(t *testing.T) {
	for _, prior := range []struct{ edition, checksum string }{
		{previousEdition, "previous-checksum"},
		{firstCampusEdition, firstCampusChecksum},
		{revisedCampusEdition, revisedCampusChecksum},
		{firstTreeholeEdition, firstTreeholeChecksum},
		{beginnerEdition, beginnerChecksum},
		{guidedEdition, guidedChecksum},
	} {
		t.Run(prior.edition, func(t *testing.T) {
			checkUpgrade(t, prior.edition, prior.checksum)
		})
	}
}

func checkUpgrade(t *testing.T, edition, checksum string) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "upgrade.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := store.Migrate(db); err != nil {
		t.Fatal(err)
	}
	baseline := func(parts ...string) string {
		return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(parts, "\x00"))))
	}
	p := Catalog{
		Chapters: []chapter{{ID: 1, Name: "Earlier chapter", Icon: "old", Description: "Earlier description", Story: "Earlier story"}},
		Levels:   []level{{ID: 1, Chapter: 1, Name: "Earlier level", Description: "Earlier level description", Tutorial: 1, guideTitle: "Earlier guide", guideBody: "Earlier walkthrough"}},
		Tasks:    []task{{ID: 1, Level: 1, Order: 1, Title: "Earlier task", body: "Earlier exercise", Criteria: json.RawMessage(`[ { "weight": 100, "criteria": "Earlier rubric" } ]`)}},
		checksum: "previous-checksum",
	}
	if _, err := p.Publish(db, true, true); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		UPDATE course_editions SET edition=?,checksum=?;
		INSERT INTO users(id,username,password,role,level,total_xp) VALUES(1,'student','fixture','EXTERNAL',1,50);
		INSERT INTO user_task_submissions(id,user_id,task_id,status,ai_review_count) VALUES(1,1,1,'APPROVED',0);
		INSERT INTO user_level_progress(level_id,user_id,status,progress,xp_earned) VALUES(1,1,'COMPLETED',100,50);
		UPDATE tasks SET description='Teacher revised' WHERE id=1`, edition, checksum); err != nil {
		t.Fatal(err)
	}
	c, l, item := &p.Chapters[0], &p.Levels[0], &p.Tasks[0]
	c.PreviousBaseline = baseline(c.Name, c.Icon, c.Description, c.Story)
	l.PreviousBaseline = baseline(l.Name, l.Description, l.guideTitle, l.guideBody)
	item.PreviousBaseline = baseline(item.Title, item.body, `[{"criteria":"Earlier rubric","weight":100}]`)
	c.CampusBaseline, l.CampusBaseline, item.CampusBaseline = c.PreviousBaseline, l.PreviousBaseline, item.PreviousBaseline
	c.TreeholeBaseline, l.TreeholeBaseline, item.TreeholeBaseline = c.PreviousBaseline, l.PreviousBaseline, item.PreviousBaseline
	c.BeginnerBaseline, l.BeginnerBaseline, item.BeginnerBaseline = c.PreviousBaseline, l.PreviousBaseline, item.PreviousBaseline
	c.GuidedBaseline, l.GuidedBaseline, item.GuidedBaseline = c.PreviousBaseline, l.PreviousBaseline, item.PreviousBaseline
	c.Name, l.Name, item.Title = "校园树洞", "匿名发帖", "发一条帖子"
	item.body, item.Criteria = "创建一条合成帖子", json.RawMessage(`[{"criteria":"树洞帖","weight":100}]`)
	item.Order, item.SourceOrder, item.BeginnerOrder = 3, 1, 1
	if edition == guidedEdition {
		item.Order = 1
	}
	p.checksum = "current-checksum"

	if _, err := p.Publish(db, false, true); err == nil || !strings.Contains(err.Error(), "task 1: content was edited") {
		t.Fatalf("expected task edit conflict, got %v", err)
	}
	var name string
	if err := db.QueryRow("SELECT name FROM maps WHERE id=1").Scan(&name); err != nil || name != "Earlier chapter" {
		t.Fatalf("chapter changed after rollback: %q %v", name, err)
	}
	if _, err := db.Exec("UPDATE tasks SET description='Earlier exercise' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Publish(db, false, true); err != nil {
		t.Fatal(err)
	}
	var title, criteria, status string
	var progress, xp, order int
	err = db.QueryRow(`SELECT t.title,t.acceptance_criteria,s.status,p.progress,p.xp_earned,t.sort_order
		FROM tasks t JOIN user_task_submissions s ON s.task_id=t.id
		JOIN user_level_progress p ON p.level_id=t.level_id AND p.user_id=s.user_id WHERE t.id=1`).
		Scan(&title, &criteria, &status, &progress, &xp, &order)
	if err != nil || title != item.Title || criteria != string(item.Criteria) || status != "APPROVED" || progress != 100 || xp != 50 || order != item.Order {
		t.Fatalf("unexpected migrated state: %q %s %s %d %d %d %v", title, criteria, status, progress, xp, order, err)
	}
}
