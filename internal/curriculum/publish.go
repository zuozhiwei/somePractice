package curriculum

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

func matches(baseline string, parts ...string) bool {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return fmt.Sprintf("%x", sum) == baseline
}

// Publish changes only course content. A rehearsal executes the same operations then rolls them back.
func (p *Catalog) Publish(db *sql.DB, initialize, apply bool) (string, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS course_editions (
		edition TEXT PRIMARY KEY, checksum TEXT NOT NULL, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP)`)
	if err != nil {
		return "", err
	}
	var checksum string
	err = tx.QueryRow("SELECT checksum FROM course_editions WHERE edition=?", Edition).Scan(&checksum)
	if err == nil {
		if checksum != p.checksum {
			return "", fmt.Errorf("edition %s already exists with a different checksum", Edition)
		}
		return "already published", nil
	}
	if err != sql.ErrNoRows {
		return "", err
	}
	var content int
	if err := tx.QueryRow(`SELECT (SELECT count(*) FROM maps)+(SELECT count(*) FROM levels)+
		(SELECT count(*) FROM tasks)+(SELECT count(*) FROM tutorials)`).Scan(&content); err != nil {
		return "", err
	}
	if initialize != (content == 0) {
		return "", fmt.Errorf("initialization requires an empty course; use -init only on a new installation")
	}
	for _, c := range p.Chapters {
		if initialize || c.Baseline == "" {
			_, err = tx.Exec(`INSERT INTO maps(id,name,icon,description,story,order_index,is_builtin) VALUES(?,?,?,?,?,?,1)`, c.ID, c.Name, c.Icon, c.Description, c.Story, c.Order)
		} else {
			var name, description, story string
			err = tx.QueryRow("SELECT name,COALESCE(description,''),COALESCE(story,'') FROM maps WHERE id=? AND is_builtin=1", c.ID).Scan(&name, &description, &story)
			if err == nil && (name != c.Name || !matches(c.Baseline, description, story)) {
				return "", fmt.Errorf("chapter %d was edited; reconcile before publishing", c.ID)
			}
			if err == nil {
				_, err = tx.Exec("UPDATE maps SET description=?,story=? WHERE id=?", c.Description, c.Story, c.ID)
			}
		}
		if err != nil {
			return "", fmt.Errorf("chapter %d: %w", c.ID, err)
		}
	}
	for _, l := range p.Levels {
		if initialize || l.GuideBaseline == "" {
			prereqs, err := json.Marshal(l.Prereqs)
			if err != nil {
				return "", err
			}
			_, err = tx.Exec(`INSERT INTO levels(id,map_id,name,description,category,tier,order_index,xp_reward,prerequisite_level_ids,is_builtin)
				VALUES(?,?,?,?,?,?,?,?,?,1)`, l.ID, l.Chapter, l.Name, l.Description, l.Category, l.Tier, l.Order, l.XP, string(prereqs))
			if err != nil {
				return "", fmt.Errorf("level %d: %w", l.ID, err)
			}
			_, err = tx.Exec(`INSERT INTO tutorials(id,level_id,title,content,sort_order,is_builtin,updated_at) VALUES(?,?,?,?,?,1,datetime('now'))`, l.Tutorial, l.ID, l.guideTitle, l.guideBody, l.TutorialOrder)
			if err != nil {
				return "", fmt.Errorf("new tutorial %d: %w", l.Tutorial, err)
			}
		} else {
			var title, body string
			originalName := l.Name
			if l.SourceName != "" {
				originalName = l.SourceName
			}
			err = tx.QueryRow(`SELECT t.title,COALESCE(t.content,'') FROM tutorials t JOIN levels l ON t.level_id=l.id
				WHERE t.id=? AND t.is_builtin=1 AND l.id=? AND l.map_id=? AND l.name=? AND l.is_builtin=1`, l.Tutorial, l.ID, l.Chapter, originalName).Scan(&title, &body)
			if err != nil {
				return "", fmt.Errorf("tutorial %d: %w", l.Tutorial, err)
			}
			if !matches(l.GuideBaseline, title, body) {
				return "", fmt.Errorf("tutorial %d was edited; reconcile before publishing", l.Tutorial)
			}
			if l.SourceName != "" {
				if _, err = tx.Exec("UPDATE levels SET name=? WHERE id=?", l.Name, l.ID); err != nil {
					return "", err
				}
			}
			if l.SourcePrereqs != nil {
				var current string
				var ids []int
				if err = tx.QueryRow("SELECT prerequisite_level_ids FROM levels WHERE id=?", l.ID).Scan(&current); err != nil {
					return "", err
				}
				if json.Unmarshal([]byte(current), &ids) != nil || !slices.Equal(ids, l.SourcePrereqs) {
					return "", fmt.Errorf("prerequisites for level %d were edited; reconcile before publishing", l.ID)
				}
				next, err := json.Marshal(l.Prereqs)
				if err != nil {
					return "", err
				}
				if _, err = tx.Exec("UPDATE levels SET prerequisite_level_ids=? WHERE id=?", string(next), l.ID); err != nil {
					return "", err
				}
			}
			if l.SourceDescription != "" {
				result, err := tx.Exec("UPDATE levels SET description=? WHERE id=? AND description=?", l.Description, l.ID, l.SourceDescription)
				if err != nil {
					return "", err
				}
				count, err := result.RowsAffected()
				if err != nil || count != 1 {
					return "", fmt.Errorf("level %d was edited; reconcile before publishing", l.ID)
				}
			}
			_, err = tx.Exec("UPDATE tutorials SET title=?,content=?,updated_at=datetime('now') WHERE id=?", l.guideTitle, l.guideBody, l.Tutorial)
			if err != nil {
				return "", err
			}
		}
	}
	for _, t := range p.Tasks {
		// Only explicit rubric revisions change an imported task's original grading criteria.
		if initialize || t.Baseline == "" {
			_, err = tx.Exec(`INSERT INTO tasks(id,level_id,title,description,sort_order,xp_reward,ai_review_enabled,acceptance_criteria,is_builtin)
				VALUES(?,?,?,?,?,?,?,?,1)`, t.ID, t.Level, t.Title, t.body, t.Order, t.XP, t.AI, string(t.Criteria))
		} else {
			var title, body, criteria string
			err = tx.QueryRow("SELECT title,COALESCE(description,''),COALESCE(acceptance_criteria,'') FROM tasks WHERE id=? AND level_id=? AND is_builtin=1", t.ID, t.Level).Scan(&title, &body, &criteria)
			if err == nil && !matches(t.Baseline, title, body, criteria) {
				return "", fmt.Errorf("task %d was edited; reconcile before publishing", t.ID)
			}
			if err == nil {
				if t.SourceTitle != "" {
					criteria = string(t.Criteria)
				}
				_, err = tx.Exec("UPDATE tasks SET title=?,description=?,acceptance_criteria=? WHERE id=?", t.Title, t.body, criteria, t.ID)
			}
		}
		if err != nil {
			return "", fmt.Errorf("task %d: %w", t.ID, err)
		}
	}
	if !apply {
		return "rehearsal passed", nil
	}
	if _, err := tx.Exec("INSERT INTO course_editions(edition,checksum) VALUES(?,?)", Edition, p.checksum); err != nil {
		return "", err
	}
	return "published", tx.Commit()
}
