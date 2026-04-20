package quiz

// RoundComplete reports whether every team in the quiz has a score recorded
// for the given round. A round with zero teams is never complete.
func RoundComplete(q Quiz, round int) bool {
	if len(q.Teams) == 0 {
		return false
	}
	for _, t := range q.Teams {
		if _, ok := t.Score(round); !ok {
			return false
		}
	}
	return true
}

// roundJustCompleted returns a round number > 0 if the transition from
// before to after newly completed at least one round. When multiple rounds
// flip, the highest round number is returned (this is what the UI wants
// for animations).
func roundJustCompleted(before, after Quiz) int {
	rounds := min(before.Config.Rounds, after.Config.Rounds)
	for r := rounds; r >= 1; r-- {
		if !RoundComplete(before, r) && RoundComplete(after, r) {
			return r
		}
	}
	return 0
}
