package api

import (
	"testing"
	"time"
)

func TestAtlasRewardsUsePersonalMappedApprovals(t *testing.T) {
	t.Setenv("SKILLTREE_TIMEZONE", "UTC")
	db, h := testApp(t)
	token := loginToken(t, h)

	type achievement struct {
		TaskID, LevelID int64
		Title           string
		ReviewedAt      *int64
	}
	var atlas struct {
		Achievements      []achievement
		TotalXp           int
		Chapters          []struct{ ID int64 }
		VisitedChapterIDs []int64
	}
	const path = "/api/maps/atlas"
	learningResponse(t, h, "GET", path, "", token, &atlas)
	if atlas.Achievements == nil || len(atlas.Achievements) != 0 || atlas.TotalXp != 0 ||
		len(atlas.Chapters) != 2 || atlas.VisitedChapterIDs == nil {
		t.Fatalf("wrong initial atlas: %+v", atlas)
	}

	learningSQL(t, db, `
		UPDATE users SET total_xp=113 WHERE id=1;
		INSERT INTO users(id,username,password,role,total_xp,level)
			SELECT 2,'other',password,'EXTERNAL',91,0 FROM users WHERE id=1;
		INSERT INTO levels(id,is_builtin,name,order_index,tier,xp_reward,prerequisite_level_ids)
			VALUES(3,0,'Unmapped level',3,1,0,'[]');
		INSERT INTO tasks(id,title,level_id,xp_reward,ai_review_enabled,is_builtin,sort_order) VALUES
			(3,'Unmapped task',3,1,0,0,1),(4,'Untethered task',NULL,1,0,0,1),
			(5,'Another student',1,1,0,0,2),(6,'Awaiting review',1,1,0,0,3),
			(7,'Approved without a legacy review date',2,1,0,0,2);
		INSERT INTO user_task_submissions(user_id,task_id,status,reviewed_at,ai_review_count) VALUES
			(1,1,'APPROVED','2026-10-01 09:00:00',0),
			(1,1,'APPROVED','2026-10-03 09:00:00',0),
			(1,1,'REJECTED','2026-10-07 09:00:00',0),
			(1,2,'APPROVED','2026-10-04 15:30:00',0),
			(1,3,'APPROVED','2026-10-05 09:00:00',0),
			(1,4,'APPROVED','2026-10-06 09:00:00',0),
			(2,5,'APPROVED','2026-10-07 09:00:00',0),
			(1,6,'PENDING',NULL,0),
			(1,7,'APPROVED',NULL,0)`)

	learningResponse(t, h, "GET", path, "", token, &atlas)
	if atlas.TotalXp != 113 || len(atlas.Achievements) != 3 {
		t.Fatalf("wrong reward count or XP: %+v", atlas)
	}
	second, first, undated := atlas.Achievements[0], atlas.Achievements[1], atlas.Achievements[2]
	newest := time.Date(2026, time.October, 4, 15, 30, 0, 0, time.UTC).UnixMilli()
	resubmitted := time.Date(2026, time.October, 3, 9, 0, 0, 0, time.UTC).UnixMilli()
	if second.TaskID != 2 || second.LevelID != 2 || second.Title != "Task two" || second.ReviewedAt == nil || *second.ReviewedAt != newest ||
		first.TaskID != 1 || first.LevelID != 1 || first.Title != "Task one" || first.ReviewedAt == nil || *first.ReviewedAt != resubmitted ||
		undated.TaskID != 7 || undated.ReviewedAt != nil {
		t.Fatalf("wrong ordered achievement details: %+v", atlas.Achievements)
	}

	var auth struct{ Token string }
	learningResponse(t, h, "POST", "/api/auth/login", `{"username":"other","password":"correct-password"}`, "", &auth)
	learningResponse(t, h, "GET", path, "", auth.Token, &atlas)
	if atlas.TotalXp != 91 || len(atlas.Achievements) != 1 || atlas.Achievements[0].TaskID != 5 {
		t.Fatalf("wrong second account rewards: %+v", atlas)
	}
}
