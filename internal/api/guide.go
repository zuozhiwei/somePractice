package api

import (
	"database/sql"
	"net/http"
	"strconv"
)

func (a *App) mentorDialog(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := strconv.ParseInt(r.URL.Query().Get("mapId"), 10, 64)
	if err != nil {
		bad(w, "地图编号无效")
		return
	}
	s, err := a.snapshot(r, user)
	if err != nil {
		fail(w)
		return
	}
	maps, err := a.summaries(r, user, s)
	if err != nil {
		fail(w)
		return
	}
	graduated := len(maps) > 0
	for _, m := range maps {
		if m["status"] != "COMPLETED" {
			graduated = false
		}
	}
	out := map[string]any{"content": "这张地图不在这里……你走错地方了。", "canSubmit": false, "mapCompleted": false, "graduated": graduated, "currentLevelId": nil, "currentLevelName": nil}
	if graduated {
		out["content"] = "🎉 你出师了！带上你的作品去闯荡江湖吧！"
		out["mapCompleted"] = true
	} else {
		for i, m := range maps {
			if m["id"].(int64) != id {
				continue
			}
			switch m["status"] {
			case "LOCKED":
				if i > 0 {
					out["content"] = "前面的地图还没通关呢，先把「" + maps[i-1]["name"].(string) + "」打完，再来这里吧。"
				}
			case "COMPLETED":
				out["content"] = "干得漂亮！这张地图你已经完全掌握了，去下一个地图吧。"
				out["mapCompleted"] = true
			default:
				out["currentLevelId"] = m["currentLevelId"]
				out["currentLevelName"] = m["currentLevelName"]
				out["canSubmit"] = true
				if n, ok := m["currentLevelName"].(string); ok {
					out["content"] = "去把「" + n + "」完成了再来找我。"
				}
			}
			break
		}
	}
	ok(w, out)
}
func (a *App) leaderboard(w http.ResponseWriter, r *http.Request, user int64) {
	page, size := 0, 20
	var err error
	if v := r.URL.Query().Get("page"); v != "" {
		page, err = strconv.Atoi(v)
		if err != nil || page < 0 || page > 1000000 {
			bad(w, "页码无效")
			return
		}
	}
	if v := r.URL.Query().Get("size"); v != "" {
		size, err = strconv.Atoi(v)
		if err != nil || size < 1 || size > 100 {
			bad(w, "数量无效")
			return
		}
	}
	tx, err := a.DB.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		fail(w)
		return
	}
	defer tx.Rollback()
	var total, leaderXP int
	if err = tx.QueryRowContext(r.Context(), `SELECT count(*), COALESCE(max(total_xp),0) FROM users`).Scan(&total, &leaderXP); err != nil {
		fail(w)
		return
	}
	rows, err := tx.QueryContext(r.Context(), `SELECT id,username,name,avatar,total_xp,COALESCE(level,0) FROM users ORDER BY total_xp DESC,id LIMIT ? OFFSET ?`, size, page*size)
	if err != nil {
		fail(w)
		return
	}
	defer rows.Close()
	content := []any{}
	for rows.Next() {
		var id int64
		var username string
		var name, avatar sql.NullString
		var xp, level int
		if rows.Scan(&id, &username, &name, &avatar, &xp, &level) != nil {
			fail(w)
			return
		}
		content = append(content, map[string]any{"rank": page*size + len(content) + 1, "userId": id, "username": username, "name": nullString(name), "avatar": nullString(avatar), "totalXp": xp, "level": level})
	}
	if rows.Err() != nil {
		fail(w)
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"content": content, "total": total, "leaderXp": leaderXP})
}
func (a *App) mission(w http.ResponseWriter, r *http.Request, user int64) {
	var chapterID int64
	if raw := r.URL.Query().Get("chapterId"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			bad(w, "章节编号无效")
			return
		}
		chapterID = id
	}
	s, err := a.snapshot(r, user)
	if err != nil {
		fail(w)
		return
	}
	maps, err := a.summaries(r, user, s)
	if err != nil {
		fail(w)
		return
	}
	var hasLevels, hasIncomplete, foundChapter bool
	var available []int64
	chapters := make(map[int64]string)
	for _, m := range maps {
		id := m["id"].(int64)
		if chapterID != 0 && chapterID != id {
			continue
		}
		foundChapter = true
		chapters[id] = m["name"].(string)
		for _, l := range s.levels {
			if !l.MapID.Valid || l.MapID.Int64 != id {
				continue
			}
			hasLevels = true
			if s.progress[l.ID].Status == "COMPLETED" {
				continue
			}
			hasIncomplete = true
			if s.unlocked(l) {
				available = append(available, l.ID)
			}
		}
	}
	if len(available) > 0 {
		tasks, err := a.learningTasks(r.Context(), user, available, true)
		if err != nil {
			fail(w)
			return
		}
		if len(tasks) > 0 {
			t := tasks[0]
			l := s.byID[t.LevelID.Int64]
			tutorials, err := a.levelTutorials(r.Context(), l.ID)
			if err != nil {
				fail(w)
				return
			}
			state := "AVAILABLE"
			if awaitingReview(t.Status) {
				state = "WAITING_REVIEW"
			}
			ok(w, map[string]any{
				"state": state,
				"level": map[string]any{"id": l.ID, "name": l.Name, "chapterName": chapters[l.MapID.Int64], "orderIndex": l.Order},
				"task":  t.response(), "tutorials": tutorials,
			})
			return
		}
	}
	if chapterID != 0 && !foundChapter {
		reply(w, 404, nil, "章节不存在")
		return
	}
	state := "BLOCKED"
	if !hasLevels {
		state = "EMPTY"
	} else if !hasIncomplete {
		state = "COMPLETE"
	} else {
		var hasTasks bool
		if err := a.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM tasks t JOIN levels l ON l.id=t.level_id
			WHERE l.map_id IS NOT NULL AND (?=0 OR l.map_id=?))`, chapterID, chapterID).Scan(&hasTasks); err != nil {
			fail(w)
			return
		}
		if !hasTasks {
			state = "EMPTY"
		}
	}
	ok(w, map[string]any{"state": state, "task": nil, "tutorials": []any{}})
}

func (a *App) teamLeaderboard(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := strconv.ParseInt(r.URL.Query().Get("teamId"), 10, 64)
	if err != nil || id <= 0 {
		bad(w, "团队无效")
		return
	}
	page, size, err := pagination(r)
	if err != nil {
		bad(w, "分页无效")
		return
	}
	var total int
	if a.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM users WHERE team_id=?`, id).Scan(&total) != nil {
		fail(w)
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `SELECT id,username,name,avatar,total_xp,COALESCE(level,0) FROM users WHERE team_id=? ORDER BY total_xp DESC,id LIMIT ? OFFSET ?`, id, size, page*size)
	if err != nil {
		fail(w)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var uid int64
		var username string
		var name, avatar sql.NullString
		var xp, level int
		if rows.Scan(&uid, &username, &name, &avatar, &xp, &level) != nil {
			fail(w)
			return
		}
		out = append(out, map[string]any{"rank": page*size + len(out) + 1, "userId": uid, "username": username, "name": nullString(name), "avatar": nullString(avatar), "totalXp": xp, "level": level})
	}
	if rows.Err() != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"content": out, "total": total})
}
