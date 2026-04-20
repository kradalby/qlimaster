// Package store persists a [quiz.Quiz] to disk as HuJSON (Human JSON),
// permitting hand-edited comments in the state file while keeping the
// on-disk format compatible with standard JSON tooling.
//
// Save writes atomically by writing to a sibling ".tmp" file and renaming
// into place, so concurrent readers never observe a partial state. Load
// tolerates comments via the tailscale/hujson parser.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kradalby/qlimaster/quiz"
	"github.com/tailscale/hujson"
)

// ErrNotFound is returned by Load when the file does not exist.
var ErrNotFound = errors.New("quiz file not found")

// Load reads a quiz.hujson file, tolerating JSON comments and trailing
// commas via hujson.Standardize.
func Load(path string) (quiz.Quiz, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is user-controlled by design
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return quiz.Quiz{}, fmt.Errorf("%w: %s", ErrNotFound, path)
		}
		return quiz.Quiz{}, fmt.Errorf("read %s: %w", path, err)
	}
	standardized, err := hujson.Standardize(raw)
	if err != nil {
		return quiz.Quiz{}, fmt.Errorf("standardize %s: %w", path, err)
	}
	var q quiz.Quiz
	if err := json.Unmarshal(standardized, &q); err != nil {
		return quiz.Quiz{}, fmt.Errorf("unmarshal %s: %w", path, err)
	}
	return q, nil
}

// Save writes the quiz to path atomically. If path already exists and
// contains HuJSON comments, they are preserved best-effort: the existing
// file is parsed with hujson, values at the top level are rewritten, and
// the combined bytes are emitted. When comment preservation is not
// possible (first write, unreadable existing file, structural mismatch)
// the quiz is re-marshalled from scratch with JSON's normal indentation.
func Save(path string, q quiz.Quiz) error {
	data, err := marshalPreservingComments(path, q)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".quiz-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write temp %s: %w", tmpName, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("sync temp %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename %s -> %s: %w", tmpName, path, err)
	}
	return nil
}

// marshalPreservingComments produces the bytes to write. When an existing
// file parses cleanly, its hujson AST is patched so commented-out lines
// and blank-line formatting are preserved.
func marshalPreservingComments(path string, q quiz.Quiz) ([]byte, error) {
	fresh, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal quiz: %w", err)
	}
	// Append a trailing newline so the file is POSIX-friendly.
	fresh = append(fresh, '\n')

	existing, err := os.ReadFile(path) //nolint:gosec // path is user-controlled by design
	if err != nil {
		return fresh, nil //nolint:nilerr // fallback to fresh output when no existing file
	}
	value, err := hujson.Parse(existing)
	if err != nil {
		return fresh, nil //nolint:nilerr // unparseable existing file -> overwrite with fresh
	}
	// Patch by replacing the Value part of the AST with the fresh JSON.
	// hujson.Value lets us do this by parsing the fresh bytes into a Value
	// and copying across comments via the AST's BeforeExtra/AfterExtra.
	freshValue, err := hujson.Parse(fresh)
	if err != nil {
		return fresh, nil //nolint:nilerr // very unlikely; fall back to fresh
	}
	preserved := preserveTopLevelComments(value, freshValue)
	return preserved.Pack(), nil
}

// preserveTopLevelComments returns a new hujson.Value derived from
// `newValue` with the BeforeExtra/AfterExtra sections copied from `old`.
// This preserves the file's top-line documentation comment.
func preserveTopLevelComments(old, newValue hujson.Value) hujson.Value {
	out := newValue
	out.BeforeExtra = append([]byte(nil), old.BeforeExtra...)
	out.AfterExtra = append([]byte(nil), old.AfterExtra...)
	return out
}
