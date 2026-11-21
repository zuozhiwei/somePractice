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

type courseSource int

const (
	importedCourse courseSource = iota
	workshopCourse
	campusCourse
	treeholeCourse
	beginnerCourse
	guidedCourse
)

func sourceCourse(tx *sql.Tx) (courseSource, error) {
	for _, prior := range []struct {
		edition, checksum string
		source            courseSource
	}{
		{guidedEdition, guidedChecksum, guidedCourse},
		{beginnerEdition, beginnerChecksum, beginnerCourse},
		{firstTreeholeEdition, firstTreeholeChecksum, treeholeCourse},
		{revisedCampusEdition, revisedCampusChecksum, campusCourse},
		{firstCampusEdition, firstCampusChecksum, campusCourse},
		{previousEdition, "", workshopCourse},
	} {
		var checksum string
		err := tx.QueryRow("SELECT checksum FROM course_editions WHERE edition=?", prior.edition).Scan(&checksum)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return importedCourse, err
		}
		if prior.checksum != "" && checksum != prior.checksum {
			return importedCourse, fmt.Errorf("edition %s has an unrecognized checksum", prior.edition)
		}
		return prior.source, nil
	}
	return importedCourse, nil
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
	if err := p.publishCourse(tx, initialize); err != nil {
		return "", err
	}
	if !apply {
		return "rehearsal passed", nil
	}
	if _, err := tx.Exec("INSERT INTO course_editions(edition,checksum) VALUES(?,?)", Edition, p.checksum); err != nil {
		return "", err
	}
	return "published", tx.Commit()
}

func (p *Catalog) publishCourse(tx *sql.Tx, initialize bool) error {
	source, err := sourceCourse(tx)
	if err != nil {
		return err
	}
	for _, c := range p.Chapters {
		if err := publishChapter(tx, c, initialize, source); err != nil {
			return fmt.Errorf("chapter %d: %w", c.ID, err)
		}
	}
	for _, l := range p.Levels {
		if err := publishLevel(tx, l, initialize, source); err != nil {
			return fmt.Errorf("level %d: %w", l.ID, err)
		}
	}
	for _, t := range p.Tasks {
		if err := publishTask(tx, t, initialize, source); err != nil {
			return fmt.Errorf("task %d: %w", t.ID, err)
		}
	}
	return nil
}

func publishLevelTier(tx *sql.Tx, l level) error {
	if l.SourceTier == 0 {
		return nil
	}
	_, err := tx.Exec("UPDATE levels SET tier=? WHERE id=? AND map_id=? AND is_builtin=1 AND tier=?", l.Tier, l.ID, l.Chapter, l.SourceTier)
	return err
}

func publishChapter(tx *sql.Tx, c chapter, initialize bool, source courseSource) error {
	if initialize || source == importedCourse && c.Baseline == "" {
		_, err := tx.Exec(`INSERT INTO maps(id,name,icon,description,story,order_index,is_builtin) VALUES(?,?,?,?,?,?,1)`, c.ID, c.Name, c.Icon, c.Description, c.Story, c.Order)
		return err
	}
	var name, icon, description, story string
	if err := tx.QueryRow(`SELECT name,COALESCE(icon,''),COALESCE(description,''),COALESCE(story,'') FROM maps WHERE id=? AND is_builtin=1`, c.ID).Scan(&name, &icon, &description, &story); err != nil {
		return err
	}
	var unchanged bool
	switch source {
	case importedCourse:
		unchanged = name == c.SourceName && matches(c.Baseline, description, story)
	case workshopCourse:
		unchanged = matches(c.PreviousBaseline, name, icon, description, story)
	case campusCourse:
		unchanged = matches(c.CampusBaseline, name, icon, description, story)
	case treeholeCourse:
		unchanged = matches(c.TreeholeBaseline, name, icon, description, story)
	case beginnerCourse:
		unchanged = matches(c.BeginnerBaseline, name, icon, description, story)
	case guidedCourse:
		unchanged = matches(c.GuidedBaseline, name, icon, description, story)
	}
	if !unchanged {
		return fmt.Errorf("content was edited; reconcile before publishing")
	}
	_, err := tx.Exec("UPDATE maps SET name=?,icon=?,description=?,story=? WHERE id=?", c.Name, c.Icon, c.Description, c.Story, c.ID)
	return err
}

func publishLevel(tx *sql.Tx, l level, initialize bool, source courseSource) error {
	prereqs, err := json.Marshal(l.Prereqs)
	if err != nil {
		return err
	}
	if initialize || source == importedCourse && l.GuideBaseline == "" {
		if _, err := tx.Exec(`INSERT INTO levels(id,map_id,name,description,category,tier,order_index,xp_reward,prerequisite_level_ids,is_builtin)
			VALUES(?,?,?,?,?,?,?,?,?,1)`, l.ID, l.Chapter, l.Name, l.Description, l.Category, l.Tier, l.Order, l.XP, string(prereqs)); err != nil {
			return err
		}
		_, err := tx.Exec(`INSERT INTO tutorials(id,level_id,title,content,sort_order,is_builtin,updated_at) VALUES(?,?,?,?,?,1,datetime('now'))`, l.Tutorial, l.ID, l.guideTitle, l.guideBody, l.TutorialOrder)
		return err
	}
	var name, description, oldPrereqs, title, body string
	if err := tx.QueryRow(`SELECT l.name,COALESCE(l.description,''),COALESCE(l.prerequisite_level_ids,'[]'),t.title,COALESCE(t.content,'')
		FROM tutorials t JOIN levels l ON t.level_id=l.id WHERE t.id=? AND t.is_builtin=1 AND l.id=? AND l.map_id=? AND l.is_builtin=1`, l.Tutorial, l.ID, l.Chapter).Scan(&name, &description, &oldPrereqs, &title, &body); err != nil {
		return err
	}
	var storedPrereqs []int
	if json.Unmarshal([]byte(oldPrereqs), &storedPrereqs) != nil {
		return fmt.Errorf("invalid prerequisites; reconcile before publishing")
	}
	originalPrereqs := l.Prereqs
	var unchanged bool
	switch source {
	case importedCourse:
		if l.SourcePrereqs != nil {
			originalPrereqs = l.SourcePrereqs
		}
		unchanged = name == l.SourceName && description == l.SourceDescription && matches(l.GuideBaseline, title, body)
	case workshopCourse:
		unchanged = matches(l.PreviousBaseline, name, description, title, body)
	case campusCourse:
		unchanged = matches(l.CampusBaseline, name, description, title, body)
	case treeholeCourse:
		unchanged = matches(l.TreeholeBaseline, name, description, title, body)
	case beginnerCourse:
		unchanged = matches(l.BeginnerBaseline, name, description, title, body)
	case guidedCourse:
		unchanged = matches(l.GuidedBaseline, name, description, title, body)
	}
	if !unchanged || !slices.Equal(storedPrereqs, originalPrereqs) {
		return fmt.Errorf("content was edited; reconcile before publishing")
	}
	if _, err := tx.Exec("UPDATE levels SET name=?,description=?,prerequisite_level_ids=? WHERE id=?", l.Name, l.Description, string(prereqs), l.ID); err != nil {
		return err
	}
	if err := publishLevelTier(tx, l); err != nil {
		return err
	}
	_, err = tx.Exec("UPDATE tutorials SET title=?,content=?,updated_at=datetime('now') WHERE id=?", l.guideTitle, l.guideBody, l.Tutorial)
	return err
}

func publishTask(tx *sql.Tx, t task, initialize bool, source courseSource) error {
	if initialize || source == importedCourse && t.Baseline == "" {
		_, err := tx.Exec(`INSERT INTO tasks(id,level_id,title,description,sort_order,xp_reward,ai_review_enabled,acceptance_criteria,is_builtin)
			VALUES(?,?,?,?,?,?,?,?,1)`, t.ID, t.Level, t.Title, t.body, t.Order, t.XP, t.AI, string(t.Criteria))
		return err
	}
	var title, body, criteria string
	var order sql.NullInt64
	if err := tx.QueryRow(`SELECT title,COALESCE(description,''),COALESCE(acceptance_criteria,''),sort_order FROM tasks WHERE id=? AND level_id=? AND is_builtin=1`, t.ID, t.Level).Scan(&title, &body, &criteria, &order); err != nil {
		return err
	}
	baseline := t.Baseline
	if source != importedCourse {
		var rubric []map[string]any
		if err := json.Unmarshal([]byte(criteria), &rubric); err != nil {
			return fmt.Errorf("invalid criteria; reconcile before publishing")
		}
		canonical, err := json.Marshal(rubric)
		if err != nil {
			return err
		}
		baseline = t.PreviousBaseline
		switch source {
		case campusCourse:
			baseline = t.CampusBaseline
		case treeholeCourse:
			baseline = t.TreeholeBaseline
		case beginnerCourse:
			baseline = t.BeginnerBaseline
		case guidedCourse:
			baseline = t.GuidedBaseline
		}
		criteria = string(canonical)
	}
	if !matches(baseline, title, body, criteria) {
		return fmt.Errorf("content was edited; reconcile before publishing")
	}
	sourceOrder := t.SourceOrder
	switch source {
	case beginnerCourse:
		sourceOrder = t.BeginnerOrder
	case guidedCourse:
		// The guided edition already uses this sequence. Preserve local reordering.
		sourceOrder = 0
	}
	if sourceOrder > 0 {
		if order.Int64 != int64(sourceOrder) {
			return fmt.Errorf("order was edited; reconcile before publishing")
		}
		order.Int64, order.Valid = int64(t.Order), true
	}
	_, err := tx.Exec("UPDATE tasks SET title=?,description=?,acceptance_criteria=?,sort_order=? WHERE id=?", t.Title, t.body, string(t.Criteria), order, t.ID)
	return err
}
