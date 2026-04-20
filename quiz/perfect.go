package quiz

// PerfectRef identifies a particular (team, round) cell that achieved a
// perfect round score. A score qualifies as perfect when it is greater than
// or equal to the configured questions-per-round value (it may be higher in
// quizzes that award bonus points).
type PerfectRef struct {
	TeamID string
	Round  int
}

// perfectRounds returns all (team, round) pairs in the quiz that are at or
// above the perfect-score threshold.
func perfectRounds(q Quiz) []PerfectRef {
	threshold := float64(q.Config.QuestionsPerRound)
	out := make([]PerfectRef, 0)
	for _, team := range q.Teams {
		for r := 1; r <= q.Config.Rounds; r++ {
			if v, ok := team.Score(r); ok && v >= threshold {
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
