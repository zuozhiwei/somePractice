package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// validatePrerequisites rejects dangling IDs, self references, and cycles.
func validatePrerequisites(db *sql.DB, target int64, mapID *int64, requested []int64) error {
	if mapID == nil || *mapID <= 0 {
		return fmt.Errorf("请选择有效地图")
	}
	var exists int
	if err := db.QueryRow(`SELECT count(*) FROM maps WHERE id=?`, *mapID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf("地图不存在")
	}
	rows, err := db.Query(`SELECT id,prerequisite_level_ids FROM levels`)
	if err != nil {
		return err
	}
	defer rows.Close()
	graph := map[int64][]int64{}
	for rows.Next() {
		var id int64
		var raw sql.NullString
		if err = rows.Scan(&id, &raw); err != nil {
			return err
		}
		var ids []int64
		if raw.Valid {
			if err = json.Unmarshal([]byte(raw.String), &ids); err != nil {
				return err
			}
		}
		graph[id] = ids
	}
	if err = rows.Err(); err != nil {
		return err
	}
	seen := map[int64]bool{}
	for _, id := range requested {
		if id <= 0 || id == target || seen[id] {
			return fmt.Errorf("前置关卡无效或重复")
		}
		seen[id] = true
		if _, ok := graph[id]; !ok {
			return fmt.Errorf("前置关卡不存在")
		}
	}
	if target <= 0 {
		return nil
	}
	if _, ok := graph[target]; !ok {
		return fmt.Errorf("关卡不存在")
	}
	graph[target] = requested
	visiting, done := map[int64]bool{}, map[int64]bool{}
	var visit func(int64) bool
	visit = func(id int64) bool {
		if visiting[id] {
			return false
		}
		if done[id] {
			return true
		}
		visiting[id] = true
		for _, next := range graph[id] {
			if !visit(next) {
				return false
			}
		}
		delete(visiting, id)
		done[id] = true
		return true
	}
	if !visit(target) {
		return fmt.Errorf("前置关系不能形成循环")
	}
	return nil
}
