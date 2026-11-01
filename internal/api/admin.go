package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (a *App) teacher(fn func(http.ResponseWriter, *http.Request, int64)) func(http.ResponseWriter, *http.Request, int64) {
	return func(w http.ResponseWriter, r *http.Request, user int64) {
		var role string
		if a.DB.QueryRowContext(r.Context(), `SELECT COALESCE(role,'') FROM users WHERE id=?`, user).Scan(&role) != nil {
			fail(w)
			return
		}
		if role != "TEACHER" {
			reply(w, 403, nil, "权限不足")
			return
		}
		fn(w, r, user)
	}
}
func (a *App) adminLevels(w http.ResponseWriter, r *http.Request, user int64) {
	s, err := a.snapshot(r, user)
	if err != nil {
		fail(w)
		return
	}
	sort.SliceStable(s.levels, func(i, j int) bool {
		a, b := s.levels[i], s.levels[j]
		if a.MapID.Valid != b.MapID.Valid {
			return a.MapID.Valid
		}
		if a.MapID.Valid && a.MapID.Int64 != b.MapID.Int64 {
			return a.MapID.Int64 < b.MapID.Int64
		}
		if a.Order != b.Order {
			return a.Order < b.Order
		}
		return a.ID < b.ID
	})
	out := []any{}
	for _, l := range s.levels {
		v := map[string]any{"id": l.ID, "mapId": nullInt(l.MapID), "name": l.Name, "description": nullString(l.Description), "category": nullString(l.Category), "tier": l.Tier, "orderIndex": l.Order, "xpReward": l.XP, "prerequisiteLevelIds": l.Prereqs}
		if r.PathValue("id") != "" {
			id, _ := idParam(r)
			if id == l.ID {
				ok(w, v)
				return
			}
		}
		out = append(out, v)
	}
	if r.PathValue("id") != "" {
		bad(w, "关卡不存在")
		return
	}
	ok(w, out)
}
func (a *App) adminSaveLevel(w http.ResponseWriter, r *http.Request, user int64) {
	a.contentMu.Lock()
	defer a.contentMu.Unlock()
	var v struct {
		Name, Description, Category string
		MapID                       *int64  `json:"mapId"`
		Tier                        int     `json:"tier"`
		Order                       int     `json:"orderIndex"`
		XP                          int     `json:"xpReward"`
		Prereqs                     []int64 `json:"prerequisiteLevelIds"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if json.NewDecoder(r.Body).Decode(&v) != nil || strings.TrimSpace(v.Name) == "" || v.XP < 0 {
		bad(w, "关卡信息无效")
		return
	}
	var targetID int64
	if r.Method != "POST" {
		var err error
		targetID, err = idParam(r)
		if err != nil {
			bad(w, "关卡编号无效")
			return
		}
	}
	if err := validatePrerequisites(a.DB, targetID, v.MapID, v.Prereqs); err != nil {
		bad(w, err.Error())
		return
	}
	b, _ := json.Marshal(v.Prereqs)
	if v.Prereqs == nil {
		b = []byte("[]")
	}
	now := time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999")
	var id int64
	var err error
	if r.Method == "POST" {
		res, e := a.DB.ExecContext(r.Context(), `INSERT INTO levels(name,description,category,tier,map_id,order_index,xp_reward,prerequisite_level_ids,is_builtin,created_by,created_at) VALUES(?,?,?,?,?,?,?,?,0,?,?)`, v.Name, v.Description, v.Category, v.Tier, v.MapID, v.Order, v.XP, string(b), user, now)
		err = e
		if e == nil {
			id, _ = res.LastInsertId()
		}
	} else {
		id = targetID
		if err == nil {
			_, err = a.DB.ExecContext(r.Context(), `UPDATE levels SET name=?,description=?,category=?,tier=?,map_id=?,order_index=?,xp_reward=?,prerequisite_level_ids=? WHERE id=?`, v.Name, v.Description, v.Category, v.Tier, v.MapID, v.Order, v.XP, string(b), id)
		}
	}
	if err != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"id": id})
}
func (a *App) adminSaveTask(w http.ResponseWriter, r *http.Request, user int64) {
	var v struct {
		Title, Description string
		LevelID            *int64          `json:"levelId"`
		XP                 int             `json:"xpReward"`
		Order              *int            `json:"sortOrder"`
		AI                 bool            `json:"aiReviewEnabled"`
		Criteria           json.RawMessage `json:"acceptanceCriteria"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if json.NewDecoder(r.Body).Decode(&v) != nil || strings.TrimSpace(v.Title) == "" || v.XP < 0 {
		bad(w, "任务信息无效")
		return
	}
	if v.LevelID == nil || *v.LevelID <= 0 {
		bad(w, "请选择有效关卡")
		return
	}
	var levelCount int
	if err := a.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM levels WHERE id=?`, *v.LevelID).Scan(&levelCount); err != nil {
		fail(w)
		return
	}
	if levelCount == 0 {
		bad(w, "关卡不存在")
		return
	}
	if len(v.Criteria) == 0 || string(v.Criteria) == "null" {
		v.Criteria = []byte("[]")
	}
	now := time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999")
	var id int64
	var err error
	if r.Method == "POST" {
		res, e := a.DB.ExecContext(r.Context(), `INSERT INTO tasks(title,description,level_id,xp_reward,sort_order,ai_review_enabled,acceptance_criteria,is_builtin,created_by,created_at) VALUES(?,?,?,?,?,?,?,0,?,?)`, v.Title, v.Description, v.LevelID, v.XP, v.Order, v.AI, string(v.Criteria), user, now)
		err = e
		if e == nil {
			id, _ = res.LastInsertId()
		}
	} else {
		id, err = idParam(r)
		if err == nil {
			_, err = a.DB.ExecContext(r.Context(), `UPDATE tasks SET title=?,description=?,level_id=?,xp_reward=?,sort_order=?,ai_review_enabled=?,acceptance_criteria=? WHERE id=?`, v.Title, v.Description, v.LevelID, v.XP, v.Order, v.AI, string(v.Criteria), id)
		}
	}
	if err != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"id": id})
}
func (a *App) adminDeleteContent(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "编号无效")
		return
	}
	table := ""
	switch {
	case strings.Contains(r.URL.Path, "/levels/"):
		table = "levels"
	case strings.Contains(r.URL.Path, "/tasks/"):
		table = "tasks"
	case strings.Contains(r.URL.Path, "/tutorials/"):
		table = "tutorials"
	}
	if table == "" {
		bad(w, "类型无效")
		return
	}
	res, err := a.DB.ExecContext(r.Context(), `DELETE FROM `+table+` WHERE id=? AND is_builtin=0`, id)
	if err != nil {
		bad(w, "无法删除：该内容仍有关联数据")
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		bad(w, "内置内容不可删除或内容不存在")
		return
	}
	ok(w, nil)
}
func (a *App) adminMoveTask(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "编号无效")
		return
	}
	direction := r.URL.Query().Get("direction")
	if direction != "up" && direction != "down" {
		bad(w, "方向无效")
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w)
		return
	}
	defer tx.Rollback()
	var lid, order sql.NullInt64
	if err = tx.QueryRow(`SELECT level_id,sort_order FROM tasks WHERE id=?`, id).Scan(&lid, &order); err != nil || !lid.Valid || !order.Valid {
		bad(w, "任务不存在或未设置排序")
		return
	}
	op, sorting := "<", "DESC"
	if direction == "down" {
		op = ">"
		sorting = "ASC"
	}
	var other, otherOrder int64
	err = tx.QueryRow(`SELECT id,sort_order FROM tasks WHERE level_id=? AND (sort_order `+op+` ? OR (sort_order=? AND id `+op+` ?)) ORDER BY sort_order `+sorting+`,id `+sorting+` LIMIT 1`, lid.Int64, order.Int64, order.Int64, id).Scan(&other, &otherOrder)
	if err == sql.ErrNoRows {
		ok(w, nil)
		return
	}
	if err != nil {
		fail(w)
		return
	}
	if otherOrder == order.Int64 {
		bad(w, "相邻任务排序相同，请先调整顺序")
		return
	}
	if _, err = tx.Exec(`UPDATE tasks SET sort_order=CASE id WHEN ? THEN ? WHEN ? THEN ? END WHERE id IN (?,?)`, id, otherOrder, other, order.Int64, id, other); err != nil {
		fail(w)
		return
	}
	if err = tx.Commit(); err != nil {
		fail(w)
		return
	}
	ok(w, nil)
}
func (a *App) tutorials(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := strconv.ParseInt(r.URL.Query().Get("levelId"), 10, 64)
	if err != nil {
		bad(w, "关卡编号无效")
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `SELECT id,title,content,sort_order,is_builtin,created_by,updated_at FROM tutorials WHERE level_id=? ORDER BY sort_order,id`, id)
	if err != nil {
		fail(w)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var tid int64
		var title string
		var content, updated sql.NullString
		var order, builtin int
		var by sql.NullInt64
		if rows.Scan(&tid, &title, &content, &order, &builtin, &by, &updated) != nil {
			fail(w)
			return
		}
		out = append(out, map[string]any{"id": tid, "levelId": id, "title": title, "content": nullString(content), "sortOrder": order, "builtin": builtin != 0, "createdBy": nullInt(by), "updatedAt": a.timestamp(updated)})
	}
	if rows.Err() != nil {
		fail(w)
		return
	}
	ok(w, out)
}
func (a *App) saveTutorial(w http.ResponseWriter, r *http.Request, user int64) {
	var v struct {
		Title, Content string
		LevelID        *int64 `json:"levelId"`
		Order          int    `json:"sortOrder"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	if json.NewDecoder(r.Body).Decode(&v) != nil || strings.TrimSpace(v.Title) == "" {
		bad(w, "教程无效")
		return
	}
	now := time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999")
	var err error
	var id int64
	if r.Method == "POST" {
		res, e := a.DB.ExecContext(r.Context(), `INSERT INTO tutorials(title,content,level_id,sort_order,is_builtin,created_by,updated_at) VALUES(?,?,?,?,0,?,?)`, v.Title, v.Content, v.LevelID, v.Order, user, now)
		err = e
		if e == nil {
			id, _ = res.LastInsertId()
		}
	} else {
		id, err = idParam(r)
		if err == nil {
			_, err = a.DB.ExecContext(r.Context(), `UPDATE tutorials SET title=?,content=?,level_id=?,sort_order=?,updated_at=? WHERE id=?`, v.Title, v.Content, v.LevelID, v.Order, now, id)
		}
	}
	if err != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"id": id})
}
func (a *App) adminUsers(w http.ResponseWriter, r *http.Request, user int64) {
	page, size, err := pagination(r)
	if err != nil {
		bad(w, "分页无效")
		return
	}
	var total int
	if a.DB.QueryRowContext(r.Context(), `SELECT count(*) FROM users`).Scan(&total) != nil {
		fail(w)
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `SELECT id FROM users ORDER BY id LIMIT ? OFFSET ?`, size, page*size)
	if err != nil {
		fail(w)
		return
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if rows.Scan(&id) != nil {
			rows.Close()
			fail(w)
			return
		}
		ids = append(ids, id)
	}
	if rows.Err() != nil {
		rows.Close()
		fail(w)
		return
	}
	rows.Close()
	out := []any{}
	for _, id := range ids {
		v, e := a.userProfile(r, id)
		if e != nil {
			fail(w)
			return
		}
		out = append(out, v)
	}
	ok(w, map[string]any{"content": out, "total": total})
}
func (a *App) adminUserUpdate(w http.ResponseWriter, r *http.Request, user int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "用户无效")
		return
	}
	if strings.HasSuffix(r.URL.Path, "/role") {
		role := r.URL.Query().Get("role")
		if role != "TEACHER" && role != "INTERNAL" && role != "EXTERNAL" {
			bad(w, "身份无效")
			return
		}
		_, err = a.DB.ExecContext(r.Context(), `UPDATE users SET role=? WHERE id=?`, role, id)
	} else {
		level, e := strconv.Atoi(r.URL.Query().Get("level"))
		if e != nil || level < 0 {
			bad(w, "级别无效")
			return
		}
		_, err = a.DB.ExecContext(r.Context(), `UPDATE users SET level=? WHERE id=?`, level, id)
	}
	if err != nil {
		fail(w)
		return
	}
	ok(w, nil)
}
func (a *App) adminSubmissions(w http.ResponseWriter, r *http.Request, user int64) {
	rows, err := a.DB.QueryContext(r.Context(), submissionSelect+`ORDER BY s.submitted_at DESC,s.id DESC LIMIT 5000`)
	if err != nil {
		fail(w)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		v, e := a.submissionRow(rows)
		if e != nil {
			fail(w)
			return
		}
		out = append(out, v)
	}
	if rows.Err() != nil {
		fail(w)
		return
	}
	ok(w, out)
}
func (a *App) adminGraph(w http.ResponseWriter, r *http.Request, user int64) {
	rows, err := a.DB.QueryContext(r.Context(), `SELECT m.mentor_id,m.apprentice_id,mu.username,COALESCE(NULLIF(mu.name,''),mu.username),COALESCE(mu.role,''),COALESCE(mu.level,0),au.username,COALESCE(NULLIF(au.name,''),au.username),COALESCE(au.role,''),COALESCE(au.level,0) FROM mentorships m JOIN users mu ON mu.id=m.mentor_id JOIN users au ON au.id=m.apprentice_id WHERE m.status='ACTIVE'`)
	if err != nil {
		fail(w)
		return
	}
	defer rows.Close()
	nodes := []any{}
	edges := []any{}
	seen := map[int64]bool{}
	for rows.Next() {
		var mid, aid int64
		var mun, mname, mrole, aun, aname, arole string
		var ml, al int
		if rows.Scan(&mid, &aid, &mun, &mname, &mrole, &ml, &aun, &aname, &arole, &al) != nil {
			fail(w)
			return
		}
		if !seen[mid] {
			nodes = append(nodes, map[string]any{"id": mid, "name": mname, "username": mun, "role": mrole, "level": ml})
			seen[mid] = true
		}
		if !seen[aid] {
			nodes = append(nodes, map[string]any{"id": aid, "name": aname, "username": aun, "role": arole, "level": al})
			seen[aid] = true
		}
		edges = append(edges, map[string]any{"mentorId": mid, "apprenticeId": aid})
	}
	if rows.Err() != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"nodes": nodes, "edges": edges})
}
func (a *App) adminInvites(w http.ResponseWriter, r *http.Request, user int64) {
	now := time.Now().In(a.zone)
	if r.Method == "DELETE" {
		id, err := idParam(r)
		if err != nil {
			bad(w, "邀请码无效")
			return
		}
		if _, err = a.DB.ExecContext(r.Context(), `DELETE FROM invite_codes WHERE id=?`, id); err != nil {
			fail(w)
			return
		}
		ok(w, nil)
		return
	}
	if r.Method == "POST" {
		var v struct {
			Code, Description string
			ExpiresAt         *int64 `json:"expiresAt"`
			TeamID            *int64 `json:"teamId"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		if json.NewDecoder(r.Body).Decode(&v) != nil || strings.TrimSpace(v.Code) == "" || len(v.Code) > 100 {
			bad(w, "邀请码无效")
			return
		}
		expires := now.Add(7 * 24 * time.Hour)
		if v.ExpiresAt != nil {
			expires = time.UnixMilli(*v.ExpiresAt).In(a.zone)
		}
		res, err := a.DB.ExecContext(r.Context(), `INSERT INTO invite_codes(active,code,created_at,created_by,description,team_id,expires_at) VALUES(1,?,?,?,?,?,?)`, v.Code, now.Format("2006-01-02 15:04:05.999999999"), user, v.Description, v.TeamID, expires.Format("2006-01-02 15:04:05.999999999"))
		if err != nil {
			bad(w, "邀请码已存在或无效")
			return
		}
		id, _ := res.LastInsertId()
		ok(w, map[string]any{"id": id})
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `SELECT id,active,code,description,team_id,created_by,created_at,expires_at FROM invite_codes WHERE expires_at>? ORDER BY created_at DESC`, now.Format("2006-01-02 15:04:05.999999999"))
	if err != nil {
		fail(w)
		return
	}
	defer rows.Close()
	out := []any{}
	for rows.Next() {
		var id int64
		var active int
		var code string
		var desc, created, expires sql.NullString
		var team, by sql.NullInt64
		if rows.Scan(&id, &active, &code, &desc, &team, &by, &created, &expires) != nil {
			fail(w)
			return
		}
		out = append(out, map[string]any{"id": id, "active": active != 0, "code": code, "description": nullString(desc), "teamId": nullInt(team), "createdBy": nullInt(by), "createdAt": a.timestamp(created), "expiresAt": a.timestamp(expires)})
	}
	if rows.Err() != nil {
		fail(w)
		return
	}
	ok(w, out)
}

func (a *App) adminDeleteUser(w http.ResponseWriter, r *http.Request, actor int64) {
	id, err := idParam(r)
	if err != nil || id == actor {
		bad(w, "不能删除当前登录用户")
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w)
		return
	}
	defer tx.Rollback()
	queries := []string{
		`DELETE FROM chat_messages WHERE conversation_id IN (SELECT id FROM conversations WHERE owner_id=?)`,
		`DELETE FROM conversation_members WHERE conversation_id IN (SELECT id FROM conversations WHERE owner_id=?)`,
		`DELETE FROM conversations WHERE owner_id=?`,
		`DELETE FROM chat_messages WHERE sender_id=?`,
		`DELETE FROM conversation_members WHERE user_id=?`,
		`DELETE FROM user_task_submissions WHERE user_id=?`,
		`UPDATE user_task_submissions SET reviewer_id=NULL WHERE reviewer_id=?`,
		`DELETE FROM mentorships WHERE mentor_id=? OR apprentice_id=?`,
		`DELETE FROM user_level_progress WHERE user_id=?`,
		`DELETE FROM user_map_progress WHERE user_id=?`,
		`DELETE FROM user_skill_mastery WHERE user_id=?`,
		`DELETE FROM user_teams WHERE user_id=?`,
		`DELETE FROM users WHERE id=?`,
	}
	for _, q := range queries {
		args := []any{id}
		if strings.Count(q, "?") == 2 {
			args = append(args, id)
		}
		if _, err = tx.ExecContext(r.Context(), q, args...); err != nil {
			fail(w)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		fail(w)
		return
	}
	ok(w, nil)
}

func (a *App) adminAssignTeam(w http.ResponseWriter, r *http.Request, actor int64) {
	id, err := idParam(r)
	if err != nil {
		bad(w, "用户无效")
		return
	}
	team, err := strconv.ParseInt(r.URL.Query().Get("teamId"), 10, 64)
	if err != nil || team <= 0 {
		bad(w, "团队无效")
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		fail(w)
		return
	}
	defer tx.Rollback()
	var count int
	if tx.QueryRow(`SELECT count(*) FROM teams WHERE id=?`, team).Scan(&count) != nil || count == 0 {
		bad(w, "团队不存在")
		return
	}
	if _, err = tx.Exec(`UPDATE users SET team_id=? WHERE id=?`, team, id); err == nil {
		_, err = tx.Exec(`DELETE FROM user_teams WHERE user_id=?`, id)
	}
	if err == nil {
		_, err = tx.Exec(`INSERT INTO user_teams(team_id,user_id,role,joined_at) VALUES(?,?,'MEMBER',?)`, team, id, time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999"))
	}
	if err != nil {
		fail(w)
		return
	}
	if tx.Commit() != nil {
		fail(w)
		return
	}
	ok(w, nil)
}
func (a *App) adminTeams(w http.ResponseWriter, r *http.Request, user int64) {
	if r.Method == "GET" {
		rows, err := a.DB.QueryContext(r.Context(), `SELECT id,name,description,created_by,created_at FROM teams ORDER BY id`)
		if err != nil {
			fail(w)
			return
		}
		defer rows.Close()
		out := []any{}
		for rows.Next() {
			var id int64
			var name string
			var desc, created sql.NullString
			var by sql.NullInt64
			if rows.Scan(&id, &name, &desc, &by, &created) != nil {
				fail(w)
				return
			}
			out = append(out, map[string]any{"id": id, "name": name, "description": nullString(desc), "createdBy": nullInt(by), "createdAt": a.timestamp(created)})
		}
		if rows.Err() != nil {
			fail(w)
			return
		}
		ok(w, out)
		return
	}
	if r.Method == "DELETE" {
		id, err := idParam(r)
		if err != nil {
			bad(w, "团队无效")
			return
		}
		tx, err := a.DB.BeginTx(r.Context(), nil)
		if err != nil {
			fail(w)
			return
		}
		defer tx.Rollback()
		for _, q := range []string{`UPDATE users SET team_id=NULL WHERE team_id=?`, `DELETE FROM user_teams WHERE team_id=?`, `UPDATE invite_codes SET team_id=NULL WHERE team_id=?`, `DELETE FROM teams WHERE id=?`} {
			if _, err = tx.Exec(q, id); err != nil {
				fail(w)
				return
			}
		}
		if tx.Commit() != nil {
			fail(w)
			return
		}
		ok(w, nil)
		return
	}
	var v struct{ Name, Description string }
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if json.NewDecoder(r.Body).Decode(&v) != nil || strings.TrimSpace(v.Name) == "" {
		bad(w, "团队名称无效")
		return
	}
	var id int64
	var err error
	if r.Method == "POST" {
		res, e := a.DB.ExecContext(r.Context(), `INSERT INTO teams(name,description,created_by,created_at) VALUES(?,?,?,?)`, v.Name, v.Description, user, time.Now().In(a.zone).Format("2006-01-02 15:04:05.999999999"))
		err = e
		if e == nil {
			id, _ = res.LastInsertId()
		}
	} else {
		id, err = idParam(r)
		if err == nil {
			_, err = a.DB.ExecContext(r.Context(), `UPDATE teams SET name=?,description=? WHERE id=?`, v.Name, v.Description, id)
		}
	}
	if err != nil {
		fail(w)
		return
	}
	ok(w, map[string]any{"id": id, "name": v.Name, "description": v.Description})
}
