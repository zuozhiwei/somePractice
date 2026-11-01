package store

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRelationEnforcement(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = Migrate(db); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO user_level_progress(level_id,user_id,status,progress,xp_earned) VALUES(123,456,'LEARNING',0,0)`); err == nil {
		t.Fatal("orphan insert accepted")
	}
	if _, err = db.Exec(`INSERT INTO maps(id,name,order_index,is_builtin) VALUES(1,'map',1,0)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO levels(id,name,map_id,is_builtin,order_index,tier,xp_reward) VALUES(1,'level',1,0,1,1,1)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`DELETE FROM maps WHERE id=1`); err == nil {
		t.Fatal("referenced parent deleted")
	}
}
func TestMigrationRejectsExistingOrphans(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = Prepare(db); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO conversation_members(conversation_id,user_id) VALUES(1,2)`); err != nil {
		t.Fatal(err)
	}
	if err = Migrate(db); err == nil || !strings.Contains(err.Error(), "orphan") {
		t.Fatalf("expected orphan check, got %v", err)
	}
}
