package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
)

type level struct {
	ID                    int64
	MapID                 sql.NullInt64
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
	s.teacher = role == "TEACHER"
	rows, err := a.DB.QueryContext(r.Context(), `SELECT id,map_id,name,description,category,tier,order_index,xp_reward,prerequisite_level_ids FROM levels ORDER BY order_index,id`)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var l level
		var raw sql.NullString
		if err = rows.Scan(&l.ID, &l.MapID, &l.Name, &l.Description, &l.Category, &l.Tier, &l.Order, &l.XP, &raw); err != nil {
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
	if s.teacher {
		return true
	}
	for _, id := range l.Prereqs {
		if s.progress[id].Status != "COMPLETED" {
			return false
		}
	}
	return true
}
func (s snapshot) levelStatus(l level) (string, int) {
	p, has := s.progress[l.ID]
	if has && p.Status != "LOCKED" {
		return p.Status, p.Progress
	}
	if s.unlocked(l) {
		return "LEARNING", 1
	}
	return "LOCKED", 0
}
func (a *App) loadTasks(r *http.Request, levelID int64) ([]task, error) {
	rows, err := a.DB.QueryContext(r.Context(), `SELECT id,title,description,acceptance_criteria,xp_reward,sort_order,ai_review_enabled,is_builtin,created_by,created_at,level_id FROM tasks WHERE level_id=? ORDER BY sort_order,id`, levelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []task{}
	for rows.Next() {
		var t task
		if err = rows.Scan(&t.ID, &t.Title, &t.Description, &t.Criteria, &t.XP, &t.Order, &t.AI, &t.Builtin, &t.CreatedBy, &t.CreatedAt, &t.LevelID); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
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
	tasks, err := a.loadTasks(r, levelID)
	if err != nil {
		fail(w)
		return
	}
	out := []any{}
	for _, t := range tasks {
		out = append(out, map[string]any{"id": t.ID, "title": t.Title, "description": nullString(t.Description), "acceptanceCriteria": criteria(t), "levelId": nullInt(t.LevelID), "xpReward": t.XP, "builtin": t.Builtin != 0, "createdBy": nullInt(t.CreatedBy), "createdAt": a.timestamp(t.CreatedAt), "sortOrder": nullInt(t.Order), "aiReviewEnabled": t.AI != 0})
	}
	ok(w, out)
}
func (a *App) summaries(r *http.Request, user int64, s snapshot) ([]map[string]any, error) {
	rows, err := a.DB.QueryContext(r.Context(), `SELECT id,name,icon,description,story,order_index FROM maps ORDER BY order_index,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	prev := true
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
			} else if currentID == nil {
				currentID = l.ID
				currentName = l.Name
			}
		}
		complete := total > 0 && done == total
		status := "IN_PROGRESS"
		if !prev && !s.teacher {
			status = "LOCKED"
		} else if complete {
			status = "COMPLETED"
		}
		if status != "IN_PROGRESS" {
			currentID = nil
			currentName = nil
		}
		out = append(out, map[string]any{"id": id, "name": name, "icon": nullString(icon), "description": nullString(desc), "story": nullString(story), "orderIndex": order, "status": status, "levelsCompleted": done, "totalLevels": total, "xpEarned": xp, "currentLevelId": currentID, "currentLevelName": currentName})
		prev = complete
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
	ok(w, m)
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
		if id == nil && v["status"] != "COMPLETED" {
			id = v["id"]
			name = v["name"]
		}
	}
	ok(w, map[string]any{"totalMaps": len(m), "totalLevels": total, "completedLevels": done, "totalXp": xp, "currentMapId": id, "currentMapName": name, "maps": m})
}
func (a *App) mapDetail(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "地图编号无效")
		return
	}
	s, err := a.snapshot(r, user)
	if err != nil {
		fail(w)
		return
	}
	summaries, err := a.summaries(r, user, s)
	if err != nil {
		fail(w)
		return
	}
	var found map[string]any
	for _, m := range summaries {
		if m["id"].(int64) == id {
			found = m
			break
		}
	}
	if found == nil {
		bad(w, "地图不存在")
		return
	}
	levels := []any{}
	for _, l := range s.levels {
		if !l.MapID.Valid || l.MapID.Int64 != id {
			continue
		}
		status := "LOCKED"
		if s.progress[l.ID].Status == "COMPLETED" {
			status = "COMPLETED"
		} else if s.unlocked(l) {
			status = "LEARNING"
		}
		var total, done int
		err = a.DB.QueryRowContext(r.Context(), `SELECT count(*), COALESCE(sum(CASE WHEN EXISTS(SELECT 1 FROM user_task_submissions s WHERE s.task_id=t.id AND s.user_id=? AND s.status='APPROVED') THEN 1 ELSE 0 END),0) FROM tasks t WHERE t.level_id=?`, user, l.ID).Scan(&total, &done)
		if err != nil {
			fail(w)
			return
		}
		levels = append(levels, map[string]any{"id": l.ID, "name": l.Name, "description": nullString(l.Description), "orderIndex": l.Order, "xpReward": l.XP, "status": status, "progress": s.progress[l.ID].Progress, "taskDone": done, "taskTotal": total})
	}
	delete(found, "xpEarned")
	delete(found, "currentLevelId")
	delete(found, "currentLevelName")
	found["levels"] = levels
	ok(w, found)
}
func (a *App) levelTree(w http.ResponseWriter, r *http.Request, user int64) {
	s, err := a.snapshot(r, user)
	if err != nil {
		fail(w)
		return
	}
	out := []any{}
	for _, l := range s.levels {
		status, progress := s.levelStatus(l)
		tasks, err := a.loadTasks(r, l.ID)
		if err != nil {
			fail(w)
			return
		}
		ts := []any{}
		for _, t := range tasks {
			ts = append(ts, map[string]any{"id": t.ID, "title": t.Title, "sortOrder": nullInt(t.Order)})
		}
		names := []string{}
		for _, id := range l.Prereqs {
			if p, ok := s.byID[id]; ok {
				names = append(names, p.Name)
			} else {
				names = append(names, "未知")
			}
		}
		out = append(out, map[string]any{"id": l.ID, "name": l.Name, "mapId": nullInt(l.MapID), "orderIndex": l.Order, "tier": l.Tier, "category": nullString(l.Category), "status": status, "progress": progress, "tasks": ts, "prerequisiteNames": names, "prerequisiteIds": l.Prereqs})
	}
	ok(w, out)
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
	rows, err := a.DB.QueryContext(r.Context(), `SELECT id,title,content,sort_order FROM tutorials WHERE level_id=? ORDER BY sort_order,id`, id)
	if err != nil {
		fail(w)
		return
	}
	tutorials := []any{}
	for rows.Next() {
		var tid int64
		var title string
		var content sql.NullString
		var order int
		if err = rows.Scan(&tid, &title, &content, &order); err != nil {
			rows.Close()
			fail(w)
			return
		}
		tutorials = append(tutorials, map[string]any{"id": tid, "title": title, "content": nullString(content), "sortOrder": order})
	}
	if rows.Err() != nil {
		rows.Close()
		fail(w)
		return
	}
	rows.Close()
	tasks, err := a.loadTasks(r, id)
	if err != nil {
		fail(w)
		return
	}
	ts := []any{}
	for _, t := range tasks {
		var subStatus string
		var feedback sql.NullString
		err = a.DB.QueryRowContext(r.Context(), `SELECT COALESCE(status,'NONE'),ai_feedback FROM user_task_submissions WHERE user_id=? AND task_id=? ORDER BY submitted_at DESC,id DESC LIMIT 1`, user, t.ID).Scan(&subStatus, &feedback)
		if err == sql.ErrNoRows {
			subStatus = "NONE"
		} else if err != nil {
			fail(w)
			return
		}
		ts = append(ts, map[string]any{"id": t.ID, "title": t.Title, "description": nullString(t.Description), "acceptanceCriteria": criteria(t), "xpReward": t.XP, "sortOrder": nullInt(t.Order), "submissionStatus": subStatus, "aiFeedback": nullString(feedback), "aiReviewEnabled": t.AI != 0})
	}
	names := []string{}
	for _, pid := range l.Prereqs {
		if p, ok := s.byID[pid]; ok {
			names = append(names, p.Name)
		} else {
			names = append(names, "未知")
		}
	}
	ok(w, map[string]any{"id": l.ID, "name": l.Name, "tier": l.Tier, "category": nullString(l.Category), "mapId": nullInt(l.MapID), "mapName": nullString(mapName), "prerequisiteNames": names, "tutorials": tutorials, "tasks": ts, "userStatus": map[string]any{"status": status, "progress": pct}})
}
