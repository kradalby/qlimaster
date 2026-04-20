// Package history maintains a running list of team names seen across
// previous quizzes so the New-Team flow can fuzzy-find likely names
// without re-typing.
//
// Two sources are combined:
//
//  1. A persistent cache at $XDG_CONFIG_HOME/qlimaster/history.hujson
//     that records the last date each name was used and how many times.
//  2. A live scan of sibling quiz folders under a configurable root,
//     which is resilient to a missing or stale cache file.
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

	"github.com/adrg/xdg"
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

// DefaultPath returns the XDG-compliant default path for the history file.
func DefaultPath() (string, error) {
	p, err := xdg.ConfigFile("qlimaster/history.hujson")
	if err != nil {
		return "", fmt.Errorf("xdg config: %w", err)
	}
	return p, nil
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
	d := date.Format("2006-01-02")
	seen := make(map[string]struct{}, len(q.Teams))
	additions := make([]Entry, 0, len(q.Teams))
	for _, t := range q.Teams {
		key := strings.ToLower(strings.TrimSpace(t.Name))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		additions = append(additions, Entry{Name: t.Name, LastSeen: d, TimesSeen: 1})
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
