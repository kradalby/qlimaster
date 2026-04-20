package quiz

import (
	"sort"
	"strings"
)

// Ranking holds a position number per team for a given quiz ordering.
// Position 1 is best (highest total). Ties share a position, the next
// distinct position is offset by the tie count (standard competition
// ranking): e.g. two teams tied for first are both 1, and the next team
// is 3.
type Ranking struct {
	// byID maps team ID to its 1-based position.
	byID map[string]int
}

// PositionOf returns the 1-based position of the team, or 0 if the team
// is not part of the ranking.
func (r Ranking) PositionOf(teamID string) int {
	return r.byID[teamID]
}

// Rank computes a Ranking for the quiz using the sum of all recorded scores
// per team. Teams are sorted descending by total; ties are resolved
// alphabetically by lowercased name (stable for equal totals). The result
// is deterministic for a given Quiz value.
//
// Ties use standard competition ranking ("1224"): two teams tied for first
// both receive position 1, and the next team receives position 3.
func Rank(q Quiz) Ranking {
	type entry struct {
		id    string
		name  string
		total float64
	}
	entries := make([]entry, 0, len(q.Teams))
	for _, t := range q.Teams {
		entries = append(entries, entry{id: t.ID, name: t.Name, total: t.Total()})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].total != entries[j].total {
			return entries[i].total > entries[j].total
		}
		return strings.ToLower(entries[i].name) < strings.ToLower(entries[j].name)
	})

	byID := make(map[string]int, len(entries))
	for i, e := range entries {
		if i > 0 && e.total == entries[i-1].total {
			byID[e.id] = byID[entries[i-1].id]
			continue
		}
		byID[e.id] = i + 1
	}
	return Ranking{byID: byID}
}

// SortByRanking returns a copy of the teams slice sorted descending by total
// (with alphabetical tiebreak). It does not modify the input quiz.
func SortByRanking(q Quiz) []Team {
	sorted := make([]Team, len(q.Teams))
	copy(sorted, q.Teams)
	sort.SliceStable(sorted, func(i, j int) bool {
		ti, tj := sorted[i].Total(), sorted[j].Total()
		if ti != tj {
			return ti > tj
		}
		return strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
	})
	return sorted
}
