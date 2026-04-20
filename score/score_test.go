package score_test

import (
	"testing"

	"github.com/kradalby/qlimaster/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  float64
	}{
		{"", 0},
		{"0", 0},
		{"1", 1},
		{"10", 10},
		{"1.", 1.5},
		{"1,", 1.5},
		{"3,", 3.5},
		{".", 0.5},
		{",", 0.5},
		{"1.5", 1.5},
		{"1,5", 1.5},
		{"0.5", 0.5},
		{"0,5", 0.5},
		{"9.5", 9.5},
		{"  2  ", 2}, // leading/trailing whitespace ignored
		{"  ,  ", 0.5},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got, err := score.Parse(tc.input, 10)
			require.NoError(t, err)
			assert.InDelta(t, tc.want, got, 1e-9)
		})
	}
}

func TestParse_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		max     float64
		wantErr error
	}{
		{"garbage", "abc", 10, score.ErrInvalid},
		{"negative", "-1", 10, score.ErrInvalid},
		{"trailing dot with non-int prefix", "1.5.", 10, score.ErrInvalid},
		{"too large", "11", 10, score.ErrOutOfRange},
		{"too large half", "10.5", 10, score.ErrOutOfRange},
		{"not half step", "1.25", 10, score.ErrNotHalfStep},
		{"not half step comma", "1,25", 10, score.ErrNotHalfStep},
		{"zero max", "1", 0, score.ErrInvalid},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := score.Parse(tc.input, tc.max)
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestParse_BoundaryValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		max   float64
		want  float64
	}{
		{"0", 10, 0},
		{"10", 10, 10},
		{"10,", 10.5, 10.5},
		{"5", 5, 5},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got, err := score.Parse(tc.input, tc.max)
			require.NoError(t, err)
			assert.InDelta(t, tc.want, got, 1e-9)
		})
	}
}

func TestFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		v    float64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{0.5, "0,5"},
		{1.5, "1,5"},
		{10.5, "10,5"},
		{2.5, "2,5"},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, score.Format(tc.v))
		})
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	for i := range 21 {
		v := float64(i) / 2
		formatted := score.Format(v)
		got, err := score.Parse(formatted, 10)
		require.NoError(t, err, "v=%v formatted=%q", v, formatted)
		assert.InDelta(t, v, got, 1e-9, "v=%v formatted=%q", v, formatted)
	}
}

// TestParse_ErrorsAreTyped ensures sentinel errors are used consistently so
// callers can discriminate with errors.Is.
func TestParse_ErrorsAreTyped(t *testing.T) {
	t.Parallel()

	_, err := score.Parse("abc", 10)
	require.Error(t, err)
	assert.ErrorIs(t, err, score.ErrInvalid)
}
