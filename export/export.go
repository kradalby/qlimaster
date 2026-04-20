// Package export renders a [quiz.Quiz] to file formats consumed by the
// outside world: CSV for quick copy/paste and long-term archives; XLSX
// for sharing a nicely-formatted sheet with non-qlimaster users.
//
// Both exporters use the same column shape:
//
//	Pos | Team | Players | R1 ... RN (interspersed with Hk checkpoints) | Total
//
// and finish with a final row of per-column averages.
package export

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/score"
)

// Header returns the header row shared by all exporters, derived from the
// quiz configuration. Round headers are "R1".."RN"; checkpoint headers are
// "H<round>".
func Header(cfg quiz.Config) []string {
	out := []string{"Pos", "Team", "Players"}
	for r := 1; r <= cfg.Rounds; r++ {
		out = append(out, fmt.Sprintf("R%d", r))
		if isCheckpoint(cfg, r) {
			out = append(out, fmt.Sprintf("H%d", r))
		}
	}
	out = append(out, "Total")
	return out
}

// Row represents a single output row. Values is parallel to the header.
type Row struct {
	Values []string
}

// BuildRows returns the body rows, already sorted into position order
// (best first), plus a final averages row.
func BuildRows(q quiz.Quiz) []Row {
	sorted := quiz.SortByRanking(q)
	ranking := quiz.Rank(q)

	rows := make([]Row, 0, len(sorted)+1)
	for _, t := range sorted {
		values := []string{
			strconv.Itoa(ranking.PositionOf(t.ID)),
			t.Name,
			t.Players,
		}
		for r := 1; r <= q.Config.Rounds; r++ {
			if v, ok := t.Score(r); ok {
				values = append(values, score.Format(v))
			} else {
				values = append(values, "")
			}
			if isCheckpoint(q.Config, r) {
				values = append(values, score.Format(quiz.Checkpoint(t, r)))
			}
		}
		values = append(values, score.Format(t.Total()))
		rows = append(rows, Row{Values: values})
	}

	rows = append(rows, averagesRow(q))
	return rows
}

func averagesRow(q quiz.Quiz) Row {
	values := []string{"avg", "", ""}
	for r := 1; r <= q.Config.Rounds; r++ {
		if avg, ok := quiz.RoundAverage(q, r); ok {
			values = append(values, formatFloat(avg))
		} else {
			values = append(values, "")
		}
		if isCheckpoint(q.Config, r) {
			if avg, ok := quiz.CheckpointAverage(q, r); ok {
				values = append(values, formatFloat(avg))
			} else {
				values = append(values, "")
			}
		}
	}
	if avg, ok := quiz.TotalAverage(q); ok {
		values = append(values, formatFloat(avg))
	} else {
		values = append(values, "")
	}
	return Row{Values: values}
}

func isCheckpoint(cfg quiz.Config, round int) bool {
	return slices.Contains(cfg.Checkpoints, round)
}

// formatFloat renders an average with two decimal places and a European
// comma separator, matching the legacy spreadsheet style.
func formatFloat(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
