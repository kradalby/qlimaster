package score_test

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/kradalby/qlimaster/score"
)

// FuzzParse verifies that Parse either returns a typed error or a value
// conforming to the documented invariants: non-negative, within [0, max],
// and a multiple of 0.5.
func FuzzParse(f *testing.F) {
	seed := []string{
		"", "0", "1", "10", "1.", "1,", ".", ",", "3,5", "3.5",
		"abc", "-1", "11", "1.25", " 2 ", "1.5.",
	}
	for _, s := range seed {
		f.Add(s, 10.0)
	}

	f.Fuzz(func(t *testing.T, s string, maxRaw float64) {
		// Bound max: Parse rejects max<=0 explicitly; for positive max we
		// clamp to something sensible so the property holds deterministically.
		if math.IsNaN(maxRaw) || math.IsInf(maxRaw, 0) {
			return
		}
		maxValue := math.Abs(maxRaw)
		if maxValue == 0 || maxValue > 1000 {
			maxValue = 10
		}
		// Force onto a half-step so the step invariant can hold.
		maxValue = math.Round(maxValue*2) / 2
		if maxValue == 0 {
			maxValue = 0.5
		}

		v, err := score.Parse(s, maxValue)
		if err != nil {
			if !errors.Is(err, score.ErrInvalid) &&
				!errors.Is(err, score.ErrOutOfRange) &&
				!errors.Is(err, score.ErrNotHalfStep) {
				t.Fatalf("untyped error for %q: %v", s, err)
			}
			return
		}
		if v < 0 || v > maxValue {
			t.Fatalf("Parse(%q, %v) = %v out of range", s, maxValue, v)
		}
		if math.Mod(v*2, 1) != 0 {
			t.Fatalf("Parse(%q, %v) = %v not a half step", s, maxValue, v)
		}
	})
}

// FuzzFormat_RoundTrip checks that formatting a valid half-step score and
// parsing the result yields the same value.
func FuzzFormat_RoundTrip(f *testing.F) {
	for i := range 21 {
		f.Add(int64(i))
	}

	f.Fuzz(func(t *testing.T, halves int64) {
		if halves < 0 || halves > 200 {
			return
		}
		v := float64(halves) / 2
		s := score.Format(v)
		if strings.ContainsAny(s, "eE") {
			t.Fatalf("Format(%v) produced scientific notation: %q", v, s)
		}
		// Must not contain '.' (European format uses ',').
		if strings.Contains(s, ".") {
			t.Fatalf("Format(%v) contains '.': %q", v, s)
		}
		got, err := score.Parse(s, 1000)
		if err != nil {
			t.Fatalf("Parse(Format(%v)=%q) error: %v", v, s, err)
		}
		if got != v {
			t.Fatalf("Parse(Format(%v)=%q) = %v", v, s, got)
		}
	})
}

// ensure strconv is referenced even if unused in future edits.
var _ = strconv.ParseFloat
