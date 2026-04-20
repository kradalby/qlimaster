// Package history maintains a running list of team names seen across
// previous quizzes so the New-Team flow can fuzzy-find likely names
// without re-typing.
//
// Two sources are combined:
//
//  1. A persistent file at <quiz-root>/history.hujson that records the
//     last date each name was used and how many times. The quiz-root
//     is the folder that holds the per-date quiz subfolders, so the
//     history lives next to them and moves with the user's sync setup.
//  2. A live scan of sibling quiz folders under the same root, which
//     is resilient to a missing or stale history file.
//
// The public API exposes a merged, deduplicated list sorted by most
// recently seen (ties broken by times-seen then by name).
package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kradalby/qlimaster/quiz"
	"github.com/tailscale/hujson"
)

// Entry is a single team-name record in the history.
type Entry struct {
	Name      string `json:"name"       hujson:"name"`
	LastSeen  string `json:"last_seen"  hujson:"last_seen"` // YYYY-MM-DD
	TimesSeen int    `json:"times_seen" hujson:"times_seen"`
}

// History is the persisted document.
type History struct {
	Version int     `json:"version" hujson:"version"`
	Teams   []Entry `json:"teams"   hujson:"teams"`
}

// DefaultPath returns the canonical history-file path for a given
// quiz-root directory (the folder that contains the per-date quiz
// subfolders). The file lives at <quizRoot>/history.hujson so it sits
// next to the individual quizzes and travels with the user's sync
// setup.
//
// An empty quizRoot is a programmer error; the caller must resolve
// the quiz root first (typically the parent of the current working
// directory).
func DefaultPath(quizRoot string) string {
	return filepath.Join(quizRoot, "history.hujson")
}

// Load reads the history file at path. A missing file is not an error:
// an empty History is returned.
func Load(path string) (History, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is user-controlled by design
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return History{Version: 1}, nil
		}
		return History{}, fmt.Errorf("read %s: %w", path, err)
	}
	standardized, err := hujson.Standardize(raw)
	if err != nil {
		return History{}, fmt.Errorf("standardize %s: %w", path, err)
	}
	var h History
	if err := json.Unmarshal(standardized, &h); err != nil {
		return History{}, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	if h.Version == 0 {
		h.Version = 1
	}
	return h, nil
}

// Save writes the history to path, creating parent directories as needed
// and using an atomic temp-file+rename cycle like [store.Save].
func Save(path string, h History) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".history-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// Merge combines multiple History values into one, deduplicating by
// case-insensitive name. The name cased in the most-recent entry is kept;
// LastSeen is the max date; TimesSeen is the sum across inputs.
func Merge(sources ...History) History {
	type acc struct {
		name      string
		lastSeen  string
		timesSeen int
	}
	byKey := make(map[string]*acc)
	for _, h := range sources {
		for _, e := range h.Teams {
			key := strings.ToLower(strings.TrimSpace(e.Name))
			if key == "" {
				continue
			}
			a, ok := byKey[key]
			if !ok {
				a = &acc{name: e.Name, lastSeen: e.LastSeen, timesSeen: e.TimesSeen}
				byKey[key] = a
				continue
			}
			a.timesSeen += e.TimesSeen
			if e.LastSeen > a.lastSeen {
				a.lastSeen = e.LastSeen
				a.name = e.Name
			}
		}
	}
	out := History{Version: 1, Teams: make([]Entry, 0, len(byKey))}
	for _, a := range byKey {
		out.Teams = append(out.Teams, Entry{
			Name:      a.name,
			LastSeen:  a.lastSeen,
			TimesSeen: a.timesSeen,
		})
	}
	SortEntries(out.Teams)
	return out
}

// SortEntries orders entries by most-recent LastSeen, ties broken by
// TimesSeen descending, then by Name ascending (case-insensitive).
func SortEntries(xs []Entry) {
	sort.SliceStable(xs, func(i, j int) bool {
		if xs[i].LastSeen != xs[j].LastSeen {
			return xs[i].LastSeen > xs[j].LastSeen
		}
		if xs[i].TimesSeen != xs[j].TimesSeen {
			return xs[i].TimesSeen > xs[j].TimesSeen
		}
		return strings.ToLower(xs[i].Name) < strings.ToLower(xs[j].Name)
	})
}

// RecordQuiz updates the history with names from a quiz seen on the given
// date. Each team is counted once per quiz.
func RecordQuiz(h History, q quiz.Quiz, date time.Time) History {
	names := make([]string, 0, len(q.Teams))
	for _, t := range q.Teams {
		names = append(names, t.Name)
	}
	return RecordNames(h, names, date)
}

// RecordNames updates the history with the supplied team names, treating
// them as a single session: duplicates within names are counted once,
// and case-insensitive matches against existing entries update
// LastSeen and bump TimesSeen by one. Empty/whitespace-only names are
// skipped.
func RecordNames(h History, names []string, date time.Time) History {
	d := date.Format("2006-01-02")
	seen := make(map[string]struct{}, len(names))
	additions := make([]Entry, 0, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		additions = append(additions, Entry{Name: name, LastSeen: d, TimesSeen: 1})
	}
	if len(additions) == 0 {
		return h
	}
	return Merge(h, History{Version: 1, Teams: additions})
}

// Names returns the history's names in their current sort order. Useful
// for feeding the fuzzy finder.
func (h History) Names() []string {
	out := make([]string, len(h.Teams))
	for i, e := range h.Teams {
		out[i] = e.Name
	}
	return out
}
