package quiz

// PerfectRef identifies a particular (team, round) cell that achieved a
// perfect round score: at or above that round's cap
// ([Config.MaxScoreForRound]).
type PerfectRef struct {
	TeamID string
	Round  int
}

// perfectRounds returns all (team, round) pairs in the quiz that are at or
// above the perfect-score threshold.
func perfectRounds(q Quiz) []PerfectRef {
	out := make([]PerfectRef, 0)

	for _, team := range q.Teams {
		for r := 1; r <= q.Config.Rounds; r++ {
			if v, ok := team.Score(r); ok && v >= q.Config.MaxScoreForRound(r) {
				out = append(out, PerfectRef{TeamID: team.ID, Round: r})
			}
		}
	}

	return out
}

// newPerfectRounds returns the set difference "after - before" of perfect
// round references, i.e. which perfect rounds were introduced by the last
// Apply.
func newPerfectRounds(before, after []PerfectRef) []PerfectRef {
	seen := make(map[PerfectRef]struct{}, len(before))
	for _, p := range before {
		seen[p] = struct{}{}
	}

	out := make([]PerfectRef, 0)

	for _, p := range after {
		if _, ok := seen[p]; !ok {
			out = append(out, p)
		}
	}

	return out
}
