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
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/kradalby/qlimaster/export"
	"github.com/kradalby/qlimaster/history"
	"github.com/kradalby/qlimaster/quiz"
	"github.com/kradalby/qlimaster/store"
	"github.com/kradalby/qlimaster/ui"
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

// newRootCommand wires the command tree.
func newRootCommand() *ff.Command {
	rootFS := ff.NewFlagSet("qlimaster")
	var (
		rounds      = rootFS.IntLong("rounds", 8, "number of rounds")
		questions   = rootFS.IntLong("questions", 10, "questions per round")
		checkpoints = rootFS.StringLong("checkpoints", "4,8",
			"comma-separated round numbers for cumulative-total columns")
		quizRoot = rootFS.StringLong("quiz-root", "",
			"root folder to scan for sibling quizzes (default: parent of CWD)")
	)

	root := &ff.Command{
		Name:      "qlimaster",
		Usage:     "qlimaster [FLAGS] [SUBCOMMAND ...]",
		ShortHelp: "pub-quiz score manager (TUI)",
		Flags:     rootFS,
		Exec: func(_ context.Context, _ []string) error {
			return runTUI(runOpts{
				rounds:      *rounds,
				questions:   *questions,
				checkpoints: *checkpoints,
				quizRoot:    *quizRoot,
			})
		},
		Subcommands: []*ff.Command{
			newVersionCommand(),
			newExportCommand(),
			newHistoryCommand(),
		},
	}
	return root
}

func newExportCommand() *ff.Command {
	fs := ff.NewFlagSet("export")
	var (
		format = fs.StringLong("format", "both", "csv | xlsx | both")
		out    = fs.StringLong("out", "", "output directory (default: CWD)")
	)
	return &ff.Command{
		Name:      "export",
		Usage:     "qlimaster export [--format=csv|xlsx|both] [--out=DIR]",
		ShortHelp: "export the quiz in the current directory",
		Flags:     fs,
		Exec: func(_ context.Context, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getwd: %w", err)
			}
			path := filepath.Join(cwd, "quiz.hujson")
			q, err := store.Load(path)
			if err != nil {
				return fmt.Errorf("load quiz: %w", err)
			}
			outDir := *out
			if outDir == "" {
				outDir = cwd
			}
			switch *format {
			case "csv":
				return exportCSV(q, outDir)
			case "xlsx":
				return exportXLSX(q, outDir)
			case "both":
				if err := exportCSV(q, outDir); err != nil {
					return err
				}
				return exportXLSX(q, outDir)
			default:
				return fmt.Errorf("unknown format %q (expected csv|xlsx|both)", *format)
			}
		},
	}
}

func exportCSV(q quiz.Quiz, dir string) error {
	target := filepath.Join(dir, "quiz.csv")
	if err := export.CSVFile(target, q); err != nil {
		return fmt.Errorf("csv: %w", err)
	}
	fmt.Println("wrote", target)
	return nil
}

func exportXLSX(q quiz.Quiz, dir string) error {
	target := filepath.Join(dir, "quiz.xlsx")
	if err := export.XLSX(target, q); err != nil {
		return fmt.Errorf("xlsx: %w", err)
	}
	fmt.Println("wrote", target)
	return nil
}

func newHistoryCommand() *ff.Command {
	return &ff.Command{
		Name:      "history",
		Usage:     "qlimaster history <subcommand>",
		ShortHelp: "manage the global team-history file",
		Flags:     ff.NewFlagSet("history"),
		Subcommands: []*ff.Command{
			newHistoryRebuildCommand(),
		},
		Exec: func(_ context.Context, _ []string) error {
			return errHistoryNeedsSub
		},
	}
}

var errHistoryNeedsSub = errors.New("history requires a subcommand")

func newHistoryRebuildCommand() *ff.Command {
	fs := ff.NewFlagSet("rebuild")
	root := fs.StringLong("quiz-root", "",
		"root folder to scan (default: parent of CWD)")
	return &ff.Command{
		Name:      "rebuild",
		Usage:     "qlimaster history rebuild [--quiz-root=DIR]",
		ShortHelp: "rescan sibling quiz folders and overwrite the history file",
		Flags:     fs,
		Exec: func(_ context.Context, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getwd: %w", err)
			}
			scanRoot := *root
			if scanRoot == "" {
				scanRoot = filepath.Dir(cwd)
			}
			scanned, err := history.Scan(scanRoot)
			if err != nil {
				return fmt.Errorf("scan %s: %w", scanRoot, err)
			}
			path := history.DefaultPath(scanRoot)
			if err := history.Save(path, scanned); err != nil {
				return fmt.Errorf("save history: %w", err)
			}
			fmt.Printf("wrote %s (%d teams)\n", path, len(scanned.Teams))
			return nil
		},
	}
}

type runOpts struct {
	rounds      int
	questions   int
	checkpoints string
	quizRoot    string
}

func runTUI(opts runOpts) error {
	cps, err := parseCheckpoints(opts.checkpoints)
	if err != nil {
		return err
	}
	cfg := quiz.Config{
		Rounds:            opts.rounds,
		QuestionsPerRound: opts.questions,
		Checkpoints:       cps,
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid quiz config: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}
	path := filepath.Join(cwd, "quiz.hujson")

	model, err := ui.New(ui.Config{
		Path:       path,
		QuizRoot:   opts.quizRoot,
		QuizConfig: cfg,
	})
	if err != nil {
		return fmt.Errorf("init ui: %w", err)
	}

	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("run program: %w", err)
	}
	return nil
}

func parseCheckpoints(s string) ([]int, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("checkpoint %q: %w", p, err)
		}
		out = append(out, v)
	}
	return out, nil
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
