package ui

import (
	"os"
	"testing"

	"github.com/adrg/xdg"
)

// TestMain isolates the UI tests from the real user's XDG config
// directory. Tests that construct a Model without an explicit
// HistoryPath rely on history.ResolvePath, which falls back to the
// XDG config path when no quiz-root ancestor is found. In a temp
// test directory no quiz-root ever qualifies, so without this isolation
// every test run would write ~/.config/qlimaster/history.hujson and
// pollute the user's real history file.
//
// xdg caches the resolved base directories at package init, so after
// overriding the env var we must call xdg.Reload to pick up the new
// value.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "qlimaster-ui-test-xdg-*")
	if err != nil {
		os.Exit(1)
	}
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	xdg.Reload()
	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}
