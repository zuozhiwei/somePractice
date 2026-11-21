package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
)

type level struct {
	ID                    int64
	MapID, Number         sql.NullInt64
	Name                  string
	Description, Category sql.NullString
	Tier, Order, XP       int
	Prereqs               []int64
}
type progress struct {
	Status       string
	Progress, XP int
}
type task struct {
	ID          int64
	Title       string
	Description sql.NullString
	Criteria    sql.NullString
	XP          int
	Order       sql.NullInt64
	AI          int
	Builtin     int
	CreatedBy   sql.NullInt64
	CreatedAt   sql.NullString
	LevelID     sql.NullInt64
}
type snapshot struct {
	levels   []level
	byID     map[int64]level
	progress map[int64]progress
	teacher  bool
}

func (a *App) snapshot(r *http.Request, user int64) (snapshot, error) {
	s := snapshot{byID: map[int64]level{}, progress: map[int64]progress{}}
	var role string
	if err := a.DB.QueryRowContext(r.Context(), `SELECT COALESCE(role,'') FROM users WHERE id=?`, user).Scan(&role); err != nil {
		return s, err
	}
	s.teacher = role == "TEACHER" || role == "SUPERADMIN"
	// Keep published lesson numbers steady when a teacher inserts custom levels.
	// Unassigned levels have no position in the course.
	rows, err := a.DB.QueryContext(r.Context(), `
		SELECT l.id,l.map_id,l.name,l.description,l.category,l.tier,l.order_index,l.xp_reward,l.prerequisite_level_ids,
			CASE WHEN m.id IS NOT NULL THEN row_number() OVER (
				PARTITION BY m.id IS NULL ORDER BY l.is_builtin DESC,m.order_index,m.id,l.order_index,l.id
			) END
		FROM levels l LEFT JOIN maps m ON m.id=l.map_id
		ORDER BY l.order_index,l.id`)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var l level
		var raw sql.NullString
		if err = rows.Scan(&l.ID, &l.MapID, &l.Name, &l.Description, &l.Category, &l.Tier, &l.Order, &l.XP, &raw, &l.Number); err != nil {
			rows.Close()
			return s, err
		}
		l.Prereqs = []int64{}
		if raw.Valid {
			if err = json.Unmarshal([]byte(raw.String), &l.Prereqs); err != nil {
				rows.Close()
				return s, err
			}
		}
		s.levels = append(s.levels, l)
		s.byID[l.ID] = l
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return s, err
	}
	rows.Close()
	rows, err = a.DB.QueryContext(r.Context(), `SELECT level_id,COALESCE(status,'LOCKED'),progress,xp_earned FROM user_level_progress WHERE user_id=?`, user)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var p progress
		if err = rows.Scan(&id, &p.Status, &p.Progress, &p.XP); err != nil {
			return s, err
		}
		s.progress[id] = p
	}
	return s, rows.Err()
}
func (s snapshot) unlocked(l level) bool {
	if s.teacher || s.progress[l.ID].Status == "COMPLETED" {
		return true
	}
	for _, id := range l.Prereqs {
		if s.progress[id].Status != "COMPLETED" {
			return false
		}
	}
	return true
}

func (s snapshot) prerequisiteNames(l level) []string {
	names := make([]string, 0, len(l.Prereqs))
	for _, id := range l.Prereqs {
		if p, ok := s.byID[id]; ok {
			names = append(names, p.Name)
		} else {
			names = append(names, "未知")
		}
	}
	return names
}
func (s snapshot) levelStatus(l level) (string, int) {
	if !s.unlocked(l) {
		return "LOCKED", 0
	}
	p, has := s.progress[l.ID]
	if has && p.Status != "LOCKED" {
		return p.Status, p.Progress
	}
	return "LEARNING", 1
}
func criteria(t task) any {
	if !t.Criteria.Valid {
		return []any{}
	}
	var v any
	if json.Unmarshal([]byte(t.Criteria.String), &v) != nil {
		return []any{}
	}
	return v
}
func (a *App) taskList(w http.ResponseWriter, r *http.Request, user int64) {
	levelID, err := strconv.ParseInt(r.URL.Query().Get("levelId"), 10, 64)
	if err != nil || levelID <= 0 {
		bad(w, "levelId 无效")
		return
	}
	tasks, err := a.learningTasks(r.Context(), user, []int64{levelID}, false)
	if err != nil {
		fail(w)
		return
	}
	out := []any{}
	for _, t := range tasks {
		row := t.response()
		row["levelId"], row["builtin"] = nullInt(t.LevelID), t.Builtin != 0
		row["createdBy"], row["createdAt"] = nullInt(t.CreatedBy), a.timestamp(t.CreatedAt)
		out = append(out, row)
	}
	okLearning(w, r, out)
}
func (a *App) summaries(r *http.Request, user int64, s snapshot) ([]map[string]any, error) {
	rows, err := a.DB.QueryContext(r.Context(), `SELECT id,name,icon,description,story,order_index FROM maps ORDER BY order_index,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var name string
		var icon, desc, story sql.NullString
		var order int
		if err = rows.Scan(&id, &name, &icon, &desc, &story, &order); err != nil {
			return nil, err
		}
		total, done, xp := 0, 0, 0
		var currentID, currentName any
		for _, l := range s.levels {
			if !l.MapID.Valid || l.MapID.Int64 != id {
				continue
			}
			total++
			p := s.progress[l.ID]
			xp += p.XP
			if p.Status == "COMPLETED" {
				done++
			} else if currentID == nil && s.unlocked(l) {
				currentID = l.ID
				currentName = l.Name
			}
		}
		complete := total > 0 && done == total
		status := "IN_PROGRESS"
		if complete {
			status = "COMPLETED"
		} else if total > 0 && done == 0 && currentID == nil {
			status = "LOCKED"
		}
		out = append(out, map[string]any{"id": id, "name": name, "icon": nullString(icon), "description": nullString(desc), "story": nullString(story), "orderIndex": order, "chapterNumber": len(out) + 1, "status": status, "levelsCompleted": done, "totalLevels": total, "xpEarned": xp, "currentLevelId": currentID, "currentLevelName": currentName})
	}
	return out, rows.Err()
}
func (a *App) maps(w http.ResponseWriter, r *http.Request, user int64) {
	s, err := a.snapshot(r, user)
	if err != nil {
		fail(w)
		return
	}
	m, err := a.summaries(r, user, s)
	if err != nil {
		fail(w)
		return
	}
	okLearning(w, r, m)
}
func (a *App) mapProgress(w http.ResponseWriter, r *http.Request, user int64) {
	s, err := a.snapshot(r, user)
	if err != nil {
		fail(w)
		return
	}
	m, err := a.summaries(r, user, s)
	if err != nil {
		fail(w)
		return
	}
	var xp int
	if err = a.DB.QueryRowContext(r.Context(), `SELECT total_xp FROM users WHERE id=?`, user).Scan(&xp); err != nil {
		fail(w)
		return
	}
	total, done := 0, 0
	var id, name any
	for _, v := range m {
		total += v["totalLevels"].(int)
		done += v["levelsCompleted"].(int)
		if id == nil && v["currentLevelId"] != nil {
			id = v["id"]
			name = v["name"]
		}
	}
	okLearning(w, r, map[string]any{"totalMaps": len(m), "totalLevels": total, "completedLevels": done, "totalXp": xp, "currentMapId": id, "currentMapName": name, "maps": m})
}
func (a *App) mapDetail(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "地图编号无效")
		return
	}
	chapters, err := a.chaptersWithLevels(r, user, id)
	if err != nil {
		fail(w)
		return
	}
	if len(chapters) == 0 {
		bad(w, "地图不存在")
		return
	}
	chapter := chapters[0]
	delete(chapter, "xpEarned")
	delete(chapter, "currentLevelId")
	delete(chapter, "currentLevelName")
	okLearning(w, r, chapter)
}

func (a *App) mapAtlas(w http.ResponseWriter, r *http.Request, user int64) {
	chapters, err := a.chaptersWithLevels(r, user, 0)
	if err != nil {
		fail(w)
		return
	}
	totalXP, achievements, err := a.atlasRewards(r.Context(), user)
	if err != nil {
		fail(w)
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), "SELECT map_id FROM atlas_visits WHERE user_id=? ORDER BY map_id", user)
	if err != nil {
		fail(w)
		return
	}
	defer rows.Close()
	visited := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			fail(w)
			return
		}
		visited = append(visited, id)
	}
	if rows.Err() != nil {
		fail(w)
		return
	}
	okLearningWithETag(w, r, user, map[string]any{
		"chapters": chapters, "visitedChapterIds": visited,
		"totalXp": totalXP, "achievements": achievements,
	})
}

type atlasAchievement struct {
	TaskID     int64  `json:"taskId"`
	LevelID    int64  `json:"levelId"`
	Title      string `json:"title"`
	ReviewedAt any    `json:"reviewedAt"`
}

func (a *App) atlasRewards(ctx context.Context, user int64) (int, []atlasAchievement, error) {
	rows, err := a.DB.QueryContext(ctx, `
		WITH approved_tasks AS (
			SELECT t.id AS task_id, l.id AS level_id, t.title, MAX(s.reviewed_at) AS reviewed_at
			FROM user_task_submissions s
			JOIN tasks t ON t.id=s.task_id
			JOIN levels l ON l.id=t.level_id
			JOIN maps m ON m.id=l.map_id
			WHERE s.user_id=? AND s.status='APPROVED'
			GROUP BY t.id, l.id, t.title
		)
		SELECT u.total_xp, ap.task_id, ap.level_id, ap.title, ap.reviewed_at
		FROM users u LEFT JOIN approved_tasks ap ON TRUE
		WHERE u.id=?
		ORDER BY ap.reviewed_at DESC, ap.task_id`, user, user)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var totalXP int
	achievements := []atlasAchievement{}
	for rows.Next() {
		var taskID, levelID sql.NullInt64
		var title, reviewedAt sql.NullString
		if err := rows.Scan(&totalXP, &taskID, &levelID, &title, &reviewedAt); err != nil {
			return 0, nil, err
		}
		if taskID.Valid {
			achievements = append(achievements, atlasAchievement{
				TaskID: taskID.Int64, LevelID: levelID.Int64, Title: title.String,
				ReviewedAt: a.timestamp(reviewedAt),
			})
		}
	}
	return totalXP, achievements, rows.Err()
}

func (a *App) visitMap(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "地图编号无效")
		return
	}
	var exists bool
	if err := a.DB.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM maps WHERE id=?)", id).Scan(&exists); err != nil {
		fail(w)
		return
	}
	if !exists {
		reply(w, 404, nil, "这个区域不存在")
		return
	}
	_, err = a.DB.ExecContext(r.Context(), "INSERT INTO atlas_visits (user_id,map_id) VALUES(?,?) ON CONFLICT DO NOTHING", user, id)
	if err != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"chapterId": id})
}

// A zero chapterID requests all regions in one read, without loading task bodies or tutorials.
func (a *App) chaptersWithLevels(r *http.Request, user, chapterID int64) ([]map[string]any, error) {
	s, err := a.snapshot(r, user)
	if err != nil {
		return nil, err
	}
	chapters, err := a.summaries(r, user, s)
	if err != nil {
		return nil, err
	}
	if chapterID != 0 {
		all := chapters
		chapters = nil
		for _, chapter := range all {
			if chapter["id"] == chapterID {
				chapters = append(chapters, chapter)
				break
			}
		}
	}
	if len(chapters) == 0 {
		return chapters, nil
	}
	type taskCount struct{ total, done int }
	counts := make(map[int64]taskCount)
	query := `SELECT t.level_id, count(*),
		sum(CASE WHEN EXISTS(SELECT 1 FROM user_task_submissions s
			WHERE s.task_id=t.id AND s.user_id=? AND s.status='APPROVED') THEN 1 ELSE 0 END)
		FROM tasks t JOIN levels l ON l.id=t.level_id WHERE l.map_id IS NOT NULL`
	args := []any{user}
	if chapterID != 0 {
		query += " AND l.map_id=?"
		args = append(args, chapterID)
	}
	rows, err := a.DB.QueryContext(r.Context(), query+" GROUP BY t.level_id", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var levelID int64
		var count taskCount
		if err := rows.Scan(&levelID, &count.total, &count.done); err != nil {
			return nil, err
		}
		counts[levelID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	levels := make(map[int64][]map[string]any, len(chapters))
	for _, chapter := range chapters {
		levels[chapter["id"].(int64)] = []map[string]any{}
	}
	for _, l := range s.levels {
		list, included := levels[l.MapID.Int64]
		if !l.MapID.Valid || !included {
			continue
		}
		status := "LOCKED"
		if s.progress[l.ID].Status == "COMPLETED" {
			status = "COMPLETED"
		} else if s.unlocked(l) {
			status = "LEARNING"
		}
		count := counts[l.ID]
		levels[l.MapID.Int64] = append(list, map[string]any{
			"id": l.ID, "name": l.Name, "description": nullString(l.Description),
			"orderIndex": l.Order, "levelNumber": nullInt(l.Number), "xpReward": l.XP, "status": status, "progress": s.progress[l.ID].Progress,
			"taskDone": count.done, "taskTotal": count.total, "prerequisiteNames": s.prerequisiteNames(l),
		})
	}
	for _, chapter := range chapters {
		chapter["levels"] = levels[chapter["id"].(int64)]
	}
	return chapters, nil
}
func (a *App) levelTree(w http.ResponseWriter, r *http.Request, user int64) {
	s, err := a.snapshot(r, user)
	if err != nil {
		fail(w)
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `SELECT level_id,id,title,sort_order FROM tasks WHERE level_id IS NOT NULL ORDER BY level_id,sort_order,id`)
	if err != nil {
		fail(w)
		return
	}
	defer rows.Close()
	byLevel := make(map[int64][]map[string]any)
	for rows.Next() {
		var lid, id int64
		var title string
		var order sql.NullInt64
		if err = rows.Scan(&lid, &id, &title, &order); err != nil {
			fail(w)
			return
		}
		byLevel[lid] = append(byLevel[lid], map[string]any{"id": id, "title": title, "sortOrder": nullInt(order)})
	}
	if rows.Err() != nil {
		fail(w)
		return
	}
	out := []any{}
	for _, l := range s.levels {
		status, progress := s.levelStatus(l)
		ts := byLevel[l.ID]
		if ts == nil {
			ts = []map[string]any{}
		}
		out = append(out, map[string]any{"id": l.ID, "name": l.Name, "mapId": nullInt(l.MapID), "orderIndex": l.Order, "levelNumber": nullInt(l.Number), "tier": l.Tier, "category": nullString(l.Category), "status": status, "progress": progress, "tasks": ts, "prerequisiteNames": s.prerequisiteNames(l), "prerequisiteIds": l.Prereqs})
	}
	okLearning(w, r, out)
}
func (a *App) levelDetail(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "关卡编号无效")
		return
	}
	s, err := a.snapshot(r, user)
	if err != nil {
		fail(w)
		return
	}
	l, exists := s.byID[id]
	if !exists {
		bad(w, "关卡不存在")
		return
	}
	if !s.unlocked(l) {
		bad(w, "前置关卡未通关")
		return
	}
	// Preserve completed progress; merely opening an unlocked level starts it.
	_, err = a.DB.ExecContext(r.Context(), `INSERT INTO user_level_progress(level_id,user_id,status,progress,xp_earned) VALUES(?,?,'LEARNING',1,0) ON CONFLICT(level_id,user_id) DO UPDATE SET status=CASE WHEN status='LOCKED' THEN 'LEARNING' ELSE status END,progress=CASE WHEN status='LOCKED' THEN 1 ELSE progress END`, id, user)
	if err != nil {
		fail(w)
		return
	}
	status, pct := s.levelStatus(l)
	if status == "LOCKED" {
		status = "LEARNING"
		pct = 1
	}
	var mapName sql.NullString
	if l.MapID.Valid {
		err = a.DB.QueryRowContext(r.Context(), `SELECT name FROM maps WHERE id=?`, l.MapID.Int64).Scan(&mapName)
		if err != nil && err != sql.ErrNoRows {
			fail(w)
			return
		}
	}
	tutorials, err := a.levelTutorials(r.Context(), id)
	if err != nil {
		fail(w)
		return
	}
	tasks, err := a.learningTasks(r.Context(), user, []int64{id}, false)
	if err != nil {
		fail(w)
		return
	}
	ts := []any{}
	for _, t := range tasks {
		ts = append(ts, t.response())
	}
	var nextLevel map[string]any
	if status == "COMPLETED" {
		for _, candidate := range s.levels {
			if slices.Contains(candidate.Prereqs, l.ID) && s.unlocked(candidate) && s.progress[candidate.ID].Status != "COMPLETED" {
				nextLevel = map[string]any{"id": candidate.ID, "name": candidate.Name}
				break
			}
		}
	}
	okLearningWithETag(w, r, user, map[string]any{
		"id": l.ID, "name": l.Name, "description": nullString(l.Description), "orderIndex": l.Order, "levelNumber": nullInt(l.Number),
		"tier": l.Tier, "category": nullString(l.Category), "mapId": nullInt(l.MapID), "mapName": nullString(mapName),
		"prerequisiteNames": s.prerequisiteNames(l), "tutorials": tutorials, "tasks": ts, "nextLevel": nextLevel,
		"userStatus": map[string]any{"status": status, "progress": pct},
	})
}
