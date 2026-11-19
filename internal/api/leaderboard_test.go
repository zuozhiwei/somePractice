package api

import (
	"encoding/json"
	"testing"
)

func TestLeaderboardGapUsesGlobalLeader(t *testing.T) {
	db, handler := testApp(t)
	if _, err := db.Exec(`INSERT INTO users(username,password,role,total_xp,level)
		VALUES ('leader','unused','INTERNAL',300,1),('second','unused','INTERNAL',140,1)`); err != nil {
		t.Fatal(err)
	}

	response := call(handler, "GET", "/api/leaderboard/global?page=1&size=1", "", loginToken(t, handler))
	var result struct {
		Data struct {
			LeaderXP int
			Total    int
			Content  []struct{ Rank, TotalXP int }
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || result.Data.LeaderXP != 300 || result.Data.Total != 3 ||
		len(result.Data.Content) != 1 || result.Data.Content[0].Rank != 2 || result.Data.Content[0].TotalXP != 140 {
		t.Fatalf("unexpected second page and leader score: %s", response.Body)
	}
}
