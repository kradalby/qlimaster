package quiz

// Change is the sealed set of mutations that can be applied to a Quiz. Each
// kind is a distinct type implementing the private isChange marker so callers
// outside this package cannot invent new change shapes.
//
// Every UI keybind that modifies state must construct a Change and pass it
// to [Apply]; Apply is the only function that mutates a Quiz.
type Change interface {
	isChange()
}

// ChangeSetScore records a score for one team in one round. Score must be
// a value accepted by score.Parse against the current
// Config.QuestionsPerRound; Apply re-validates for safety.
type ChangeSetScore struct {
	TeamID string
	Round  int
	Score  float64
}

func (ChangeSetScore) isChange() {}

// ChangeClearScore removes any recorded score for a team in a round.
type ChangeClearScore struct {
	TeamID string
	Round  int
}

func (ChangeClearScore) isChange() {}

// ChangeAddTeam appends a new team with the given name and (optional) players
// string. A stable team id is generated inside Apply; it is not required to
// be supplied here.
type ChangeAddTeam struct {
	Name    string
	Players string
}

func (ChangeAddTeam) isChange() {}

// ChangeRenameTeam updates the display name of a team.
type ChangeRenameTeam struct {
	TeamID string
	Name   string
}

func (ChangeRenameTeam) isChange() {}

// ChangeSetPlayers updates the free-text players column of a team.
type ChangeSetPlayers struct {
	TeamID  string
	Players string
}

func (ChangeSetPlayers) isChange() {}

// ChangeDeleteTeam removes a team from the quiz.
type ChangeDeleteTeam struct {
	TeamID string
}

func (ChangeDeleteTeam) isChange() {}

// ChangeSetConfig replaces the quiz Config. Reducing Rounds drops any scores
// recorded for rounds that no longer exist.
type ChangeSetConfig struct {
	Config Config
}

func (ChangeSetConfig) isChange() {}
