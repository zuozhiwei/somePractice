package api

import (
	"testing"

	"skilltree-go/internal/curriculum"
)

func TestCourseNumbersAcrossLearningViews(t *testing.T) {
	db, h := testApp(t)
	learningSQL(t, db, `DELETE FROM tutorials; DELETE FROM tasks; DELETE FROM levels; DELETE FROM maps;
		UPDATE users SET role='TEACHER' WHERE id=1`)
	pack, err := curriculum.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pack.Publish(db, true, true); err != nil {
		t.Fatal(err)
	}
	learningSQL(t, db, `
		INSERT INTO maps(id,name,order_index,is_builtin) VALUES(100,'Custom',0,0);
		INSERT INTO levels(id,name,map_id,order_index,tier,xp_reward,is_builtin) VALUES
			(100,'Early extension',100,1,1,0,0),
			(101,'Inside chapter',3,0,1,0,0),
			(102,'Unassigned',NULL,0,1,0,0)`)
	token := loginToken(t, h)
	type numberFields struct {
		ID, OrderIndex int
		LevelNumber    *int
	}
	type chapterFields struct {
		ID, ChapterNumber int
		Levels            []numberFields
	}
	checkLevel := func(l numberFields) {
		t.Helper()
		if l.ID == 102 {
			if l.LevelNumber != nil {
				t.Fatalf("unassigned level has sequence number %d", *l.LevelNumber)
			}
			return
		}
		want := l.ID
		if l.ID == 100 {
			want = 47
		} else if l.ID == 101 {
			want = 48
		}
		if l.LevelNumber == nil || *l.LevelNumber != want {
			t.Fatalf("level %d does not display number %d: %+v", l.ID, want, l)
		}
	}
	var atlas struct{ Chapters []chapterFields }
	learningResponse(t, h, "GET", "/api/maps/atlas", "", token, &atlas)
	if len(atlas.Chapters) != 9 {
		t.Fatalf("expected eight built-in chapters and one extension, got %d", len(atlas.Chapters))
	}
	for i, chapter := range atlas.Chapters {
		if chapter.ChapterNumber != i+1 {
			t.Fatalf("chapter %d does not follow display order: %+v", chapter.ID, chapter)
		}
		for _, l := range chapter.Levels {
			checkLevel(l)
		}
	}
	var chapter chapterFields
	learningResponse(t, h, "GET", "/api/maps/3", "", token, &chapter)
	if chapter.ChapterNumber != 4 {
		t.Fatalf("chapter detail number differs from the atlas: %+v", chapter)
	}
	for _, l := range chapter.Levels {
		checkLevel(l)
	}
	var tree []numberFields
	learningResponse(t, h, "GET", "/api/levels/tree", "", token, &tree)
	for _, l := range tree {
		checkLevel(l)
	}
	var detail numberFields
	learningResponse(t, h, "GET", "/api/levels/13", "", token, &detail)
	checkLevel(detail)
	if detail.OrderIndex != 1 {
		t.Fatalf("chapter-local ordering changed: %+v", detail)
	}
}
