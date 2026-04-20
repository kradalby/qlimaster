// Command qlimaster is a keyboard-driven TUI for managing pub-quiz scores.
//
// The default invocation, with no subcommand, launches the full-screen TUI in
// the current working directory (expected to be a per-quiz folder, e.g.
// 2026-04-14). Subcommands are provided for batch operations such as
// exporting the current quiz or rebuilding the team-name history cache.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

// version is stamped at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) || errors.Is(err, ff.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "qlimaster:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	root := newRootCommand()
	if err := root.Parse(args); err != nil {
		if errors.Is(err, ff.ErrHelp) {
			fmt.Fprintln(os.Stderr, ffhelp.Command(root))
			return fmt.Errorf("help: %w", err)
		}
		return fmt.Errorf("parse: %w", err)
	}
	if err := root.Run(ctx); err != nil {
		return fmt.Errorf("run: %w", err)
	}
	return nil
}

// newRootCommand wires the command tree. Subcommands are stubs for now; the
// root command is a placeholder that prints the version and exits until the
// TUI is implemented.
func newRootCommand() *ff.Command {
	rootFS := ff.NewFlagSet("qlimaster")
	var (
		_ = rootFS.IntLong("rounds", 8, "number of rounds")
		_ = rootFS.IntLong("questions", 10, "questions per round")
		_ = rootFS.StringLong("checkpoints", "4,8",
			"comma-separated round numbers for cumulative-total columns")
		_ = rootFS.StringLong("quiz-root", "",
			"root folder to scan for sibling quizzes (default: parent of CWD)")
	)

	root := &ff.Command{
		Name:      "qlimaster",
		Usage:     "qlimaster [FLAGS] [SUBCOMMAND ...]",
		ShortHelp: "pub-quiz score manager (TUI)",
		Flags:     rootFS,
		Exec: func(_ context.Context, _ []string) error {
			fmt.Printf("qlimaster %s\n", version)
			fmt.Println("TUI not yet implemented. Coming soon.")
			return nil
		},
		Subcommands: []*ff.Command{
			newVersionCommand(),
		},
	}
	return root
}

func newVersionCommand() *ff.Command {
	fs := ff.NewFlagSet("version")
	return &ff.Command{
		Name:      "version",
		Usage:     "qlimaster version",
		ShortHelp: "print qlimaster version",
		Flags:     fs,
		Exec: func(_ context.Context, _ []string) error {
			fmt.Println(version)
			return nil
		},
	}
}
