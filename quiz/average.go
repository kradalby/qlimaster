package quiz

// RoundAverage returns the mean score recorded for the given round across
// all teams that have a value for it. If no team has a score for the round
// it returns 0 and ok=false.
func RoundAverage(q Quiz, round int) (float64, bool) {
	var sum float64
	var n int
	for _, t := range q.Teams {
		if v, ok := t.Score(round); ok {
			sum += v
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}

// CheckpointAverage returns the mean cumulative total at the given round
// across every team in the quiz. Missing per-round scores count as zero
// (matching the existing spreadsheet behaviour).
func CheckpointAverage(q Quiz, round int) (float64, bool) {
	if len(q.Teams) == 0 {
		return 0, false
	}
	var sum float64
	for _, t := range q.Teams {
		sum += Checkpoint(t, round)
	}
	return sum / float64(len(q.Teams)), true
}

// TotalAverage is the mean Total across all teams.
func TotalAverage(q Quiz) (float64, bool) {
	if len(q.Teams) == 0 {
		return 0, false
	}
	var sum float64
	for _, t := range q.Teams {
		sum += t.Total()
	}
	return sum / float64(len(q.Teams)), true
}
