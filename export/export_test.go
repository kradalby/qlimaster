package export_test

import (
	"bytes"
	"encoding/csv"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kradalby/qlimaster/export"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestHeader(t *testing.T) {
	t.Parallel()

	cfg := quiz.Config{Rounds: 4, QuestionsPerRound: 10, Checkpoints: []int{2, 4}}
	got := export.Header(cfg)
	want := []string{"Pos", "Team", "Players", "R1", "R2", "H2", "R3", "R4", "H4", "Total"}
	assert.Equal(t, want, got)
}

func TestCSV_Shape(t *testing.T) {
	t.Parallel()

	q := sampleQuiz(t)
	var buf bytes.Buffer
	require.NoError(t, export.CSV(&buf, q))

	rows, err := csv.NewReader(&buf).ReadAll()
	require.NoError(t, err)

	// header + teams + averages row
	require.Len(t, rows, 1+len(q.Teams)+1)
	expectedCols := len(export.Header(q.Config))
	for _, r := range rows {
		assert.Len(t, r, expectedCols)
	}
	// Last row begins with "avg".
	assert.Equal(t, "avg", rows[len(rows)-1][0])
}

func TestCSV_OrderedByRanking(t *testing.T) {
	t.Parallel()

	q := sampleQuiz(t)
	var buf bytes.Buffer
	require.NoError(t, export.CSV(&buf, q))

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// Winner (Underpuppies with higher total) should appear before the
	// lower-scoring team.
	var underIdx, rookiesIdx int
	for i, l := range lines {
		if strings.Contains(l, "Underpuppies") {
			underIdx = i
		}
		if strings.Contains(l, "The rookies") {
			rookiesIdx = i
		}
	}
	assert.Less(t, underIdx, rookiesIdx)
}

func TestXLSX_ReadBack(t *testing.T) {
	t.Parallel()

	q := sampleQuiz(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "out.xlsx")
	require.NoError(t, export.XLSX(out, q))

	f, err := excelize.OpenFile(out)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	require.Contains(t, sheets, "Quiz")

	rows, err := f.GetRows("Quiz")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 3) // header + at least one team + avg
	assert.Equal(t, "Pos", rows[0][0])
	assert.Equal(t, "avg", rows[len(rows)-1][0])
}

func sampleQuiz(t *testing.T) quiz.Quiz {
	t.Helper()
	q := quiz.Quiz{
		Version: 1,
		Config:  quiz.Config{Rounds: 2, QuestionsPerRound: 10, Checkpoints: []int{2}},
		Teams: []quiz.Team{
			{ID: "a", Name: "The rookies", Scores: map[string]float64{"1": 5, "2": 3}},
			{ID: "b", Name: "Underpuppies", Scores: map[string]float64{"1": 8, "2": 9}},
		},
	}
	return q
}
