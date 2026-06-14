// Package quiz is the domain model and pure logic for a pub-quiz session.
//
// All state mutations flow through [Apply], the single entry point that
// validates a [Change], updates the quiz, and recomputes derived state
// (totals, ranks, perfect-round flags, averages). The UI layer wraps Apply
// with its save+animate side effects and is forbidden from mutating a Quiz
// directly; that invariant is enforced by keeping the mutable paths in this
// package.
package quiz

import (
	"fmt"
	"strconv"
	"time"
)

// Config describes the shape of the quiz (number of rounds, questions per
// round, and which rounds should show a cumulative-total checkpoint
// column).
type Config struct {
	Rounds            int            `hujson:"rounds"              json:"rounds"`
	QuestionsPerRound int            `hujson:"questions_per_round" json:"questions_per_round"`
	RoundMaxPoints    map[string]int `hujson:"round_max_points"    json:"round_max_points"`
	// MaxPoints is the highest score a team may be awarded in one round,
	// decoupled from QuestionsPerRound so a round can be worth more points
	// than it has questions (e.g. 10 questions at 2 points each). A zero
	// value means "fall back to QuestionsPerRound" so quiz files written
	// before this field existed keep their original cap.
	MaxPoints   int   `hujson:"max_points"  json:"max_points"`
	Checkpoints []int `hujson:"checkpoints" json:"checkpoints"`
}

// MaxScore returns the effective per-round score cap: MaxPoints when set,
// otherwise QuestionsPerRound for backward compatibility with older files.
func (c Config) MaxScore() float64 {
	if c.MaxPoints > 0 {
		return float64(c.MaxPoints)
	}

	return float64(c.QuestionsPerRound)
}

// MaxScoreForRound returns a round-specific score cap when configured,
// otherwise it falls back to the quiz-wide MaxScore.
func (c Config) MaxScoreForRound(round int) float64 {
	if pts, ok := c.RoundMaxPoints[strconv.Itoa(round)]; ok && pts > 0 {
		return float64(pts)
	}

	return c.MaxScore()
}

// DefaultConfig returns the standard Grandcafe de Burcht setup: 8 rounds of
// 10 questions each with cumulative-total checkpoints after rounds 4 and 8.
func DefaultConfig() Config {
	return Config{
		Rounds:            8,
		QuestionsPerRound: 10,
		MaxPoints:         20,
		Checkpoints:       []int{4, 8},
	}
}

// Validate checks that a Config describes a reasonable quiz. Rounds and
// QuestionsPerRound must be positive and not absurdly large; each
// checkpoint must be within [1, Rounds] and the list must be sorted and
// contain no duplicates.
func (c Config) Validate() error {
	if c.Rounds < 1 || c.Rounds > 50 {
		return fmt.Errorf("%w: rounds %d not in [1, 50]", ErrInvalidConfig, c.Rounds)
	}

	if c.QuestionsPerRound < 1 || c.QuestionsPerRound > 100 {
		return fmt.Errorf("%w: questions_per_round %d not in [1, 100]", ErrInvalidConfig, c.QuestionsPerRound)
	}
	// MaxPoints == 0 means "use QuestionsPerRound" (legacy files); any
	// explicit value must be positive and not absurdly large.
	if c.MaxPoints < 0 || c.MaxPoints > 1000 {
		return fmt.Errorf("%w: max_points %d not in [0, 1000]", ErrInvalidConfig, c.MaxPoints)
	}

	if err := c.validateRoundMaxPoints(); err != nil {
		return err
	}

	prev := 0

	for _, cp := range c.Checkpoints {
		if cp < 1 || cp > c.Rounds {
			return fmt.Errorf("%w: checkpoint %d not in [1, %d]", ErrInvalidConfig, cp, c.Rounds)
		}

		if cp <= prev {
			return fmt.Errorf("%w: checkpoints must be sorted ascending and unique: %v", ErrInvalidConfig, c.Checkpoints)
		}

		prev = cp
	}

	return nil
}

// validateRoundMaxPoints checks that every per-round override is keyed by a
// round in [1, Rounds] and caps the score in [1, 1000].
func (c Config) validateRoundMaxPoints() error {
	for k, pts := range c.RoundMaxPoints {
		r, err := strconv.Atoi(k)
		if err != nil || r < 1 || r > c.Rounds {
			return fmt.Errorf("%w: round_max_points key %q not in [1, %d]", ErrInvalidConfig, k, c.Rounds)
		}

		if pts < 1 || pts > 1000 {
			return fmt.Errorf("%w: round_max_points[%s] %d not in [1, 1000]", ErrInvalidConfig, k, pts)
		}
	}

	return nil
}

// Team holds a single team's identity and per-round scores.
//
// Scores is keyed by round number as a string so HuJSON serialisation is
// stable and round-keys survive JSON's stringly-typed object semantics
// without needing a custom marshaller. A missing key means the team has no
// score recorded for that round yet (treated as zero for totalling but
// displayed as blank).
type Team struct {
	ID      string             `hujson:"id"      json:"id"`
	Name    string             `hujson:"name"    json:"name"`
	Players string             `hujson:"players" json:"players"`
	Scores  map[string]float64 `hujson:"scores"  json:"scores"`
}

// Quiz is the top-level state of a quiz session. The fields are exported
// so tests and the store package can introspect them, but all external
// mutation must happen through [Apply].
type Quiz struct {
	Version int       `hujson:"version" json:"version"`
	Created time.Time `hujson:"created" json:"created"`
	Config  Config    `hujson:"config"  json:"config"`
	Teams   []Team    `hujson:"teams"   json:"teams"`
}

// New returns a fresh quiz with the supplied config and no teams.
func New(cfg Config) Quiz {
	return Quiz{
		Version: 1,
		Created: time.Now().UTC(),
		Config:  cfg,
		Teams:   []Team{},
	}
}

// Score returns the score recorded for a team in a round, and ok=true when
// a value was explicitly set (including 0); if no value exists, ok is
// false.
func (t Team) Score(round int) (float64, bool) {
	v, ok := t.Scores[roundKey(round)]

	return v, ok
}

// Total returns the sum of all recorded scores for the team.
func (t Team) Total() float64 {
	var total float64
	for _, v := range t.Scores {
		total += v
	}

	return total
}

// roundKey converts a round number to the string used as a map key.
func roundKey(round int) string {
	return strconv.Itoa(round)
}

// FindTeam returns a pointer into q.Teams to the team with the given id, or
// nil if no such team exists. The returned pointer is only valid until the
// next Apply call.
func (q *Quiz) FindTeam(id string) *Team {
	for i := range q.Teams {
		if q.Teams[i].ID == id {
			return &q.Teams[i]
		}
	}

	return nil
}

// HasTeamNamed reports whether any team in the quiz has the given name
// (case-insensitive).
func (q *Quiz) HasTeamNamed(name string) bool {
	for i := range q.Teams {
		if equalFold(q.Teams[i].Name, name) {
			return true
		}
	}

	return false
}
