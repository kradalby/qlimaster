package fuzzy_test

import (
	"testing"

	"github.com/kradalby/qlimaster/fuzzy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDo_EmptyQueryPassesAll(t *testing.T) {
	t.Parallel()

	items := []string{"Alpha", "Beta", "Gamma"}
	got := fuzzy.Do("", items)
	require.Len(t, got, len(items))
	for i, m := range got {
		assert.Equal(t, items[i], m.Item)
		assert.Zero(t, m.Score)
		assert.Empty(t, m.Positions)
	}
}

func TestDo_CaseInsensitive(t *testing.T) {
	t.Parallel()

	items := []string{"Dark Horse", "dark matter", "DARK WEB"}
	got := fuzzy.Do("dark", items)
	require.Len(t, got, 3)
}

func TestDo_ReturnsRelevantFirst(t *testing.T) {
	t.Parallel()

	items := []string{"The rookies", "Durham dummies", "Dark Horse", "Underpuppies"}
	got := fuzzy.Do("und", items)
	require.NotEmpty(t, got)
	assert.Equal(t, "Underpuppies", got[0].Item)
}

func TestDo_DiacriticInsensitive(t *testing.T) {
	t.Parallel()

	items := []string{"Café Royale", "Cafe Royale"}
	got := fuzzy.Do("cafe", items)
	require.Len(t, got, 2)
}

func TestDo_Deterministic(t *testing.T) {
	t.Parallel()

	items := []string{"Alpha", "Alphabet", "Alphanumeric", "Apple"}
	first := fuzzy.Do("alp", items)
	for range 20 {
		got := fuzzy.Do("alp", items)
		require.Equal(t, first, got)
	}
}

func TestDo_NoMatches(t *testing.T) {
	t.Parallel()

	items := []string{"Alpha", "Beta"}
	got := fuzzy.Do("zzzzz", items)
	assert.Empty(t, got)
}

func TestDo_PositionsAreIndices(t *testing.T) {
	t.Parallel()

	items := []string{"Underpuppies"}
	got := fuzzy.Do("und", items)
	require.Len(t, got, 1)
	require.Len(t, got[0].Positions, 3)
	// Positions should be strictly increasing (sorted) and in range.
	prev := -1
	for _, p := range got[0].Positions {
		assert.Greater(t, p, prev)
		prev = p
		assert.Less(t, p, len(items[0]))
	}
}

func TestDo_TiebreakAlphabetical(t *testing.T) {
	t.Parallel()

	// Query "alpha" matches these identically but different rankings; ensure
	// the tiebreak sort by lower-cased name is stable.
	items := []string{"zeta-alpha", "Alpha-zeta"}
	got := fuzzy.Do("alpha", items)
	require.Len(t, got, 2)
	// With equal scores, lower-cased alphabetical order places "Alpha-zeta"
	// first.
	if got[0].Score == got[1].Score {
		assert.Equal(t, "Alpha-zeta", got[0].Item)
	}
}
