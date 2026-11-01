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
	var total int
	if err = a.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM users`).Scan(&total); err != nil {
		fail(w)
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `SELECT id,username,name,avatar,total_xp,COALESCE(level,0) FROM users ORDER BY total_xp DESC,id LIMIT ? OFFSET ?`, size, page*size)
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
	ok(w, map[string]any{"content": content, "total": total})
}
func (a *App) mission(w http.ResponseWriter, r *http.Request, user int64) {
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
	allCompleted := len(s.levels) > 0
	var taskCount int
	if err = a.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM tasks WHERE level_id IS NOT NULL`).Scan(&taskCount); err != nil {
		fail(w)
		return
	}
	hasTasks := taskCount > 0
	for _, m := range maps {
		for _, l := range s.levels {
			if !l.MapID.Valid || l.MapID.Int64 != m["id"].(int64) {
				continue
			}
			if s.progress[l.ID].Status == "COMPLETED" {
				continue
			}
			allCompleted = false
			if !s.unlocked(l) {
				continue
			}
			tasks, err := a.loadTasks(r, l.ID)
			if err != nil {
				fail(w)
				return
			}
			if len(tasks) > 0 {
				hasTasks = true
			}
			var selected *task
			var selStatus string
			var feedback sql.NullString
			for _, t := range tasks {
				var approved int
				if err = a.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM user_task_submissions WHERE user_id=? AND task_id=? AND status='APPROVED'`, user, t.ID).Scan(&approved); err != nil {
					fail(w)
					return
				}
				if approved > 0 {
					continue
				}
				var status string
				var fb sql.NullString
				err = a.DB.QueryRowContext(r.Context(), `SELECT COALESCE(status,'NONE'),ai_feedback FROM user_task_submissions WHERE user_id=? AND task_id=? ORDER BY submitted_at DESC,id DESC LIMIT 1`, user, t.ID).Scan(&status, &fb)
				if err == sql.ErrNoRows {
					status = "NONE"
				} else if err != nil {
					fail(w)
					return
				}
				if selected == nil {
					copy := t
					selected = &copy
					selStatus = status
					feedback = fb
				}
				if status != "PENDING" && status != "AI_REVIEWED" {
					copy := t
					selected = &copy
					selStatus = status
					feedback = fb
					break
				}
			}
			if selected != nil {
				rows, err := a.DB.QueryContext(r.Context(), `SELECT id,title,content,sort_order FROM tutorials WHERE level_id=? ORDER BY sort_order,id`, l.ID)
				if err != nil {
					fail(w)
					return
				}
				tutorials := []any{}
				for rows.Next() {
					var id int64
					var title string
					var content sql.NullString
					var ord int
					if rows.Scan(&id, &title, &content, &ord) != nil {
						rows.Close()
						fail(w)
						return
					}
					tutorials = append(tutorials, map[string]any{"id": id, "title": title, "content": nullString(content), "sortOrder": ord})
				}
				if rows.Err() != nil {
					rows.Close()
					fail(w)
					return
				}
				rows.Close()
				state := "AVAILABLE"
				if selStatus == "PENDING" || selStatus == "AI_REVIEWED" {
					state = "WAITING_REVIEW"
				}
				t := *selected
				ok(w, map[string]any{"state": state, "task": map[string]any{"id": t.ID, "title": t.Title, "description": nullString(t.Description), "acceptanceCriteria": criteria(t), "xpReward": t.XP, "sortOrder": nullInt(t.Order), "submissionStatus": selStatus, "aiFeedback": nullString(feedback), "aiReviewEnabled": t.AI != 0}, "tutorials": tutorials})
				return
			}
		}
	}
	state := "BLOCKED"
	if allCompleted {
		state = "COMPLETE"
	} else if len(s.levels) == 0 || !hasTasks {
		state = "EMPTY"
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
