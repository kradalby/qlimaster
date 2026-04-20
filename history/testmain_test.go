package history_test

import (
	"os"
	"testing"

	"github.com/adrg/xdg"
)

// TestMain redirects XDG_CONFIG_HOME to a per-run temp directory so
// TestResolvePath_FallsBackToXDG (and any future test that hits
// xdg.ConfigFile) does not create directories under the real user's
// ~/.config.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "qlimaster-history-test-xdg-*")
	if err != nil {
		os.Exit(1)
	}
	_ = os.Setenv("XDG_CONFIG_HOME", tmp)
	xdg.Reload()
	code := m.Run()
	_ = os.RemoveAll(tmp)
	os.Exit(code)
}
