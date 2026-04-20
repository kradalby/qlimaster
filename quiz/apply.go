package quiz

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/kradalby/qlimaster/score"
)

// Result describes the side-effects the UI should react to after a
// successful Apply. All fields are safe to read; a Change that does not
// mutate state (e.g. setting a score to its existing value) leaves
// Mutated=false and all other fields zero.
type Result struct {
	// Mutated is true when Apply changed the quiz from its previous value.
	Mutated bool

	// RoundJustCompleted is the round number (1-based) that became fully
	// scored as a result of this Apply, or 0 if none.
	RoundJustCompleted int

	// NewPerfectRounds is the set of (team, round) cells that newly
	// reached the perfect-round threshold with this Apply.
	NewPerfectRounds []PerfectRef

	// ReRanked is true when the team order changed as a result of this
	// Apply. Rankings are only recomputed when a round completes, when the
	// config changes, or when a team is added or removed.
	ReRanked bool

	// WinnerDecided is true when, after this Apply, every round is complete
	// and a single team holds position 1 alone.
	WinnerDecided bool
}

// Errors returned by Apply.
var (
	ErrUnknownTeam   = errors.New("unknown team")
	ErrDuplicateTeam = errors.New("team name already used")
	ErrInvalidRound  = errors.New("round out of range")
	ErrInvalidChange = errors.New("invalid change")
	ErrEmptyTeamName = errors.New("team name must not be empty")
	ErrInvalidConfig = errors.New("invalid config")
)

// Apply is the single entry point for every mutation to a Quiz. It
// validates the Change, produces a new Quiz value (pure functional, the
// input is never modified), recomputes derived state, and reports what
// changed in the returned Result.
//
// The UI wraps this function in ui.Model.apply, which is the only place
// where the result is persisted to disk and animations are triggered.
func Apply(q Quiz, c Change) (Quiz, Result, error) {
	before := deepCopy(q)
	after := deepCopy(q)

	if err := applyChange(&after, c); err != nil {
		return q, Result{}, err
	}

	beforePerfect := perfectRounds(before)
	afterPerfect := perfectRounds(after)
	newPerfect := newPerfectRounds(beforePerfect, afterPerfect)

	roundDone := roundJustCompleted(before, after)
	reRanked := shouldRerank(c, roundDone)

	res := Result{
		Mutated:            !equalQuiz(before, after),
		RoundJustCompleted: roundDone,
		NewPerfectRounds:   newPerfect,
		ReRanked:           reRanked,
		WinnerDecided:      winnerDecided(after),
	}
	return after, res, nil
}

// shouldRerank returns true when the change may alter the team ordering.
//
// Per the design, sorting is deferred until a round completes (to avoid
// reshuffling rows while the host is still entering a round). Team list
// membership changes and config changes always re-rank.
func shouldRerank(c Change, roundJustDone int) bool {
	switch c.(type) {
	case ChangeAddTeam, ChangeDeleteTeam, ChangeSetConfig:
		return true
	}
	return roundJustDone > 0
}

// winnerDecided returns true when every round is complete and a single
// team holds the best total by itself.
func winnerDecided(q Quiz) bool {
	if len(q.Teams) == 0 {
		return false
	}
	for r := 1; r <= q.Config.Rounds; r++ {
		if !RoundComplete(q, r) {
			return false
		}
	}
	rank := Rank(q)
	ones := 0
	for _, t := range q.Teams {
		if rank.PositionOf(t.ID) == 1 {
			ones++
		}
	}
	return ones == 1
}

// applyChange dispatches on the concrete Change type and mutates q.
func applyChange(q *Quiz, c Change) error {
	switch ch := c.(type) {
	case ChangeSetScore:
		return applySetScore(q, ch)
	case ChangeClearScore:
		return applyClearScore(q, ch)
	case ChangeAddTeam:
		return applyAddTeam(q, ch)
	case ChangeRenameTeam:
		return applyRenameTeam(q, ch)
	case ChangeSetPlayers:
		return applySetPlayers(q, ch)
	case ChangeDeleteTeam:
		return applyDeleteTeam(q, ch)
	case ChangeSetConfig:
		return applySetConfig(q, ch)
	default:
		return fmt.Errorf("%w: %T", ErrInvalidChange, c)
	}
}

func applySetScore(q *Quiz, c ChangeSetScore) error {
	team := q.FindTeam(c.TeamID)
	if team == nil {
		return fmt.Errorf("%w: %q", ErrUnknownTeam, c.TeamID)
	}
	if c.Round < 1 || c.Round > q.Config.Rounds {
		return fmt.Errorf("%w: %d not in [1, %d]", ErrInvalidRound, c.Round, q.Config.Rounds)
	}
	// Revalidate the score against the current config; Parse is the
	// authoritative validator and rejecting here keeps the data file clean
	// even if the UI misbehaved.
	if _, err := score.Parse(score.Format(c.Score), float64(q.Config.QuestionsPerRound)); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidChange, err)
	}
	if team.Scores == nil {
		team.Scores = map[string]float64{}
	}
	team.Scores[roundKey(c.Round)] = c.Score
	return nil
}

func applyClearScore(q *Quiz, c ChangeClearScore) error {
	team := q.FindTeam(c.TeamID)
	if team == nil {
		return fmt.Errorf("%w: %q", ErrUnknownTeam, c.TeamID)
	}
	if c.Round < 1 || c.Round > q.Config.Rounds {
		return fmt.Errorf("%w: %d not in [1, %d]", ErrInvalidRound, c.Round, q.Config.Rounds)
	}
	delete(team.Scores, roundKey(c.Round))
	return nil
}

func applyAddTeam(q *Quiz, c ChangeAddTeam) error {
	name := strings.TrimSpace(c.Name)
	if name == "" {
		return ErrEmptyTeamName
	}
	if q.HasTeamNamed(name) {
		return fmt.Errorf("%w: %q", ErrDuplicateTeam, name)
	}
	q.Teams = append(q.Teams, Team{
		ID:      newTeamID(),
		Name:    name,
		Players: strings.TrimSpace(c.Players),
		Scores:  map[string]float64{},
	})
	return nil
}

func applyRenameTeam(q *Quiz, c ChangeRenameTeam) error {
	team := q.FindTeam(c.TeamID)
	if team == nil {
		return fmt.Errorf("%w: %q", ErrUnknownTeam, c.TeamID)
	}
	name := strings.TrimSpace(c.Name)
	if name == "" {
		return ErrEmptyTeamName
	}
	// Allow renaming to the same name (case-only change, or identity) but
	// reject collisions with any other team.
	for i := range q.Teams {
		if q.Teams[i].ID == c.TeamID {
			continue
		}
		if equalFold(q.Teams[i].Name, name) {
			return fmt.Errorf("%w: %q", ErrDuplicateTeam, name)
		}
	}
	team.Name = name
	return nil
}

func applySetPlayers(q *Quiz, c ChangeSetPlayers) error {
	team := q.FindTeam(c.TeamID)
	if team == nil {
		return fmt.Errorf("%w: %q", ErrUnknownTeam, c.TeamID)
	}
	team.Players = strings.TrimSpace(c.Players)
	return nil
}

func applyDeleteTeam(q *Quiz, c ChangeDeleteTeam) error {
	for i := range q.Teams {
		if q.Teams[i].ID == c.TeamID {
			q.Teams = append(q.Teams[:i], q.Teams[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("%w: %q", ErrUnknownTeam, c.TeamID)
}

func applySetConfig(q *Quiz, c ChangeSetConfig) error {
	if err := c.Config.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}
	// Drop scores outside the new round range.
	for i := range q.Teams {
		for k := range q.Teams[i].Scores {
			r := parseRoundKey(k)
			if r < 1 || r > c.Config.Rounds {
				delete(q.Teams[i].Scores, k)
			}
		}
	}
	q.Config = c.Config
	return nil
}

// deepCopy returns a value-equal copy of q with independent maps and slices
// so the input Quiz to Apply is never mutated.
func deepCopy(q Quiz) Quiz {
	out := q
	out.Config.Checkpoints = append([]int(nil), q.Config.Checkpoints...)
	out.Teams = make([]Team, len(q.Teams))
	for i, t := range q.Teams {
		copied := t
		copied.Scores = make(map[string]float64, len(t.Scores))
		maps.Copy(copied.Scores, t.Scores)
		out.Teams[i] = copied
	}
	return out
}

// equalQuiz reports whether two quizzes are value-equal. Used by Apply to
// determine whether anything actually changed.
func equalQuiz(a, b Quiz) bool {
	if a.Version != b.Version || !a.Created.Equal(b.Created) {
		return false
	}
	if a.Config.Rounds != b.Config.Rounds ||
		a.Config.QuestionsPerRound != b.Config.QuestionsPerRound {
		return false
	}
	if len(a.Config.Checkpoints) != len(b.Config.Checkpoints) {
		return false
	}
	for i, cp := range a.Config.Checkpoints {
		if cp != b.Config.Checkpoints[i] {
			return false
		}
	}
	if len(a.Teams) != len(b.Teams) {
		return false
	}
	for i := range a.Teams {
		if !equalTeam(a.Teams[i], b.Teams[i]) {
			return false
		}
	}
	return true
}

func equalTeam(a, b Team) bool {
	if a.ID != b.ID || a.Name != b.Name || a.Players != b.Players {
		return false
	}
	if len(a.Scores) != len(b.Scores) {
		return false
	}
	for k, v := range a.Scores {
		if bv, ok := b.Scores[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

// newTeamID returns a short, URL-safe, random ID. Team IDs are never shown
// to the user; they exist so sort-order changes don't invalidate
// references held by the UI.
func newTeamID() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand failing is essentially impossible; fall back to a
		// placeholder rather than panicking.
		return "t_fallback"
	}
	return "t_" + hex.EncodeToString(buf)
}

// parseRoundKey inverts roundKey; invalid input returns 0 so the caller can
// treat it as out-of-range.
func parseRoundKey(k string) int {
	r, err := strconv.Atoi(k)
	if err != nil {
		return 0
	}
	return r
}
