package quiz

// Checkpoint returns the cumulative total score for a team up to and
// including the given round.
func Checkpoint(t Team, round int) float64 {
	var total float64
	for r := 1; r <= round; r++ {
		if v, ok := t.Score(r); ok {
			total += v
		}
	}
	return total
}
