// Package score parses and formats quiz round scores.
//
// Scores are stored as float64 in steps of 0.5 in the closed interval
// [0, max]. Parse implements the shorthand input rules described below; all
// numeric input fields in the UI are routed through it so shorthand and
// validation live in one place.
//
// Shorthand rules:
//
//	input   value
//	"1"     1.0
//	"1."    1.5      whole number followed by '.' or ',' adds 0.5
//	"1,"    1.5
//	"3,"    3.5
//	"."     0.5      a bare '.' or ',' is 0.5
//	","     0.5
//	""      0.0
//	"1.5"   1.5      standard decimal fallback (both '.' and ',' work)
//	"1,5"   1.5
//	"10"    10.0
//
// Values outside [0, max] or that do not parse as one of the shapes above
// return a non-nil error.
package score

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalid is returned for input that does not match any shorthand rule.
var ErrInvalid = errors.New("invalid score")

// ErrOutOfRange is returned when a parsed value is outside [0, max].
var ErrOutOfRange = errors.New("score out of range")

// ErrNotHalfStep is returned when a parsed value is not a multiple of 0.5.
var ErrNotHalfStep = errors.New("score must be a multiple of 0.5")

// Parse converts raw user input into a score, honouring the shorthand rules
// documented on the package. maxValue is the maximum allowed value,
// inclusive (typically the configured questions-per-round). maxValue must
// be greater than 0.
func Parse(input string, maxValue float64) (float64, error) {
	if maxValue <= 0 {
		return 0, fmt.Errorf("%w: max must be positive", ErrInvalid)
	}

	s := strings.TrimSpace(input)
	if s == "" {
		return 0, nil
	}

	v, err := parseValue(s)
	if err != nil {
		return 0, err
	}

	return validate(v, maxValue)
}

// parseValue applies the shorthand and decimal rules but not range or step
// validation. Kept separate so tests can exercise the shape rules in
// isolation.
func parseValue(s string) (float64, error) {
	// Normalise European decimal separators by treating ',' identically to
	// '.'. This also makes the "1," = 1.5 rule a simple suffix test below.
	normalised := strings.ReplaceAll(s, ",", ".")

	// Bare "." => 0.5.
	if normalised == "." {
		return 0.5, nil
	}

	// Trailing "." means "half" when the prefix parses as a non-negative
	// integer: "1." -> 1.5, "3." -> 3.5, "10." -> 10.5.
	if strings.HasSuffix(normalised, ".") && strings.Count(normalised, ".") == 1 {
		prefix := strings.TrimSuffix(normalised, ".")
		if prefix == "" {
			return 0.5, nil
		}
		n, err := strconv.ParseUint(prefix, 10, 32)
		if err == nil {
			return float64(n) + 0.5, nil
		}
		// Fall through so "abc." errors out below.
	}

	// Standard decimal parse. Reject negative values explicitly.
	if strings.HasPrefix(normalised, "-") {
		return 0, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	v, err := strconv.ParseFloat(normalised, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalid, s)
	}
	return v, nil
}

// validate enforces range and half-step rules.
func validate(v, maxValue float64) (float64, error) {
	if v < 0 || v > maxValue {
		return 0, fmt.Errorf("%w: %v not in [0, %v]", ErrOutOfRange, v, maxValue)
	}
	// 0.5 step: twice the value must be (approximately) an integer.
	twice := v * 2
	rounded := float64(int64(twice + 0.5))
	if diff := twice - rounded; diff > 1e-9 || diff < -1e-9 {
		return 0, fmt.Errorf("%w: %v", ErrNotHalfStep, v)
	}
	return rounded / 2, nil
}

// Format renders a score in the European style used by the old spreadsheet:
// integer values print without a decimal, halves print with a "," separator.
//
//	0     -> "0"
//	1     -> "1"
//	2.5   -> "2,5"
//	10    -> "10"
func Format(v float64) string {
	twice := int64(v*2 + 0.5)
	if twice%2 == 0 {
		return strconv.FormatInt(twice/2, 10)
	}
	return fmt.Sprintf("%d,5", twice/2)
}
