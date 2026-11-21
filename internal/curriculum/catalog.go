// Package curriculum publishes the versioned teaching material separately from server startup.
package curriculum

import (
	"crypto/sha256"
	"embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

//go:embed catalog.json lessons/*.md
var materials embed.FS

const Edition = "campus-treehole-2026-11-v4"

const previousEdition = "fullstack-workshop-2026-11"
const firstCampusEdition = "campus-pocket-2026-11"
const firstCampusChecksum = "a591def14d4cda05da20cfc5dcc7122f101255b6cdb168da7bbd090d0a8c987d"
const revisedCampusEdition = "campus-pocket-2026-11-v2"
const revisedCampusChecksum = "fa67df24755080f3aef99ecc285557b58868bb1a6993cec1e6edfce0d60b3447"
const firstTreeholeEdition = "campus-treehole-2026-11"
const firstTreeholeChecksum = "41570d3ede02181a07e99c1290042e914538fb01ff4be74fcf2246955f66efde"

const beginnerEdition = "campus-treehole-2026-11-v2"
const beginnerChecksum = "4d1bd12c2972f1f16ddab43e78493088f48d95caf3223cd6b94c56234cdca275"

const guidedEdition = "campus-treehole-2026-11-v3"
const guidedChecksum = "1ec0bcd821932cf2d7e8a032605f0f8ffe97637c2b8a46bb3ad9d710b61b7974"

type chapter struct {
	ID, Order                                                                            int
	Name, SourceName, Icon, Description, Story, Baseline                                 string
	PreviousBaseline, CampusBaseline, TreeholeBaseline, BeginnerBaseline, GuidedBaseline string
}

type level struct {
	ID, Chapter, Tier, SourceTier, Order, XP, Tutorial, TutorialOrder                    int
	Name, SourceName, Description, SourceDescription, Category, GuideBaseline            string
	PreviousBaseline, CampusBaseline, TreeholeBaseline, BeginnerBaseline, GuidedBaseline string
	Prereqs, SourcePrereqs                                                               []int
	guideTitle, guideBody                                                                string
}

type task struct {
	ID, Level, Order, SourceOrder, BeginnerOrder, XP, AI                                                  int
	Title, Baseline, PreviousBaseline, CampusBaseline, TreeholeBaseline, BeginnerBaseline, GuidedBaseline string
	Criteria                                                                                              json.RawMessage
	body                                                                                                  string
}

type Catalog struct {
	Chapters []chapter
	Levels   []level
	Tasks    []task
	checksum string
}

var marker = regexp.MustCompile(`(?m)^## 任务 ([0-9]+) \| ([^\n]+)$`)

func Load() (*Catalog, error) {
	raw, err := materials.ReadFile("catalog.json")
	if err != nil {
		return nil, err
	}
	pack := new(Catalog)
	if err := json.Unmarshal(raw, pack); err != nil {
		return nil, err
	}
	digest := sha256.New()
	digest.Write([]byte(strings.ReplaceAll(string(raw), "\r\n", "\n")))
	tasks := make(map[int]*task)
	for i := range pack.Tasks {
		t := &pack.Tasks[i]
		if tasks[t.ID] != nil {
			return nil, fmt.Errorf("duplicate task %d", t.ID)
		}
		tasks[t.ID] = t
	}
	for i := range pack.Levels {
		l := &pack.Levels[i]
		raw, err := materials.ReadFile(fmt.Sprintf("lessons/%02d.md", l.ID))
		if err != nil {
			return nil, err
		}
		body := strings.ReplaceAll(string(raw), "\r\n", "\n")
		fmt.Fprintf(digest, "\x00%d\x00%s", l.ID, body)
		title, body, ok := strings.Cut(body, "\n")
		if !ok || !strings.HasPrefix(title, "# ") {
			return nil, fmt.Errorf("level %d has no lesson title", l.ID)
		}
		l.guideTitle = strings.TrimSpace(strings.TrimPrefix(title, "# "))
		sections := marker.FindAllStringSubmatchIndex(body, -1)
		if len(sections) == 0 {
			return nil, fmt.Errorf("level %d has no task instructions", l.ID)
		}
		l.guideBody = strings.TrimSpace(body[:sections[0][0]])
		if l.guideBody == "" {
			return nil, fmt.Errorf("level %d has no walkthrough", l.ID)
		}
		for n, s := range sections {
			id, _ := strconv.Atoi(body[s[2]:s[3]])
			t := tasks[id]
			heading := strings.TrimSpace(body[s[4]:s[5]])
			if t == nil || t.Level != l.ID || t.body != "" || heading != t.Title {
				return nil, fmt.Errorf("unexpected task %d (%s) in level %d", id, heading, l.ID)
			}
			end := len(body)
			if n+1 < len(sections) {
				end = sections[n+1][0]
			}
			t.body = strings.TrimSpace(body[s[1]:end])
			if t.body == "" {
				return nil, fmt.Errorf("task %d has no instructions", id)
			}
		}
	}
	for _, t := range tasks {
		if t.body == "" {
			return nil, fmt.Errorf("missing task %d", t.ID)
		}
	}
	pack.checksum = fmt.Sprintf("%x", digest.Sum(nil))
	return pack, nil
}
