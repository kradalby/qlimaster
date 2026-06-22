# qlimaster

Keyboard-driven, full-screen terminal UI for hosting pub quizzes — a
spreadsheet replacement, built for the quiz at Grandcafe de Burcht in Leiden.

- Live score table that fills the terminal, re-sorted worst-to-best each round.
- Fast entry: round, fuzzy team picker, score, repeat.
- Fuzzy team lookup against past quizzes; perfect-round highlighting.
- Configurable cumulative-total checkpoint columns and per-round point caps.
- Always-saved HuJSON state; CSV/XLSX export.

## Install

Prebuilt binaries (linux/darwin/windows, amd64/arm64) are published on every
push to `main` as the rolling **continuous** prerelease:

https://github.com/kradalby/qlimaster/releases

Or build from source:

```
go install github.com/kradalby/qlimaster/cmd/qlimaster@latest
nix run github:kradalby/qlimaster      # with Nix + flakes
```

## Usage

Run in the quiz's working directory; a `quiz.hujson` is created if absent:

```
cd ~/quiz/$(date +%Y-%m-%d)
qlimaster
```

Override the shape of a new quiz with flags (ignored once `quiz.hujson` exists):

```
qlimaster --rounds=12 --questions=24 --points=24 --checkpoints=4,8,12
```

Subcommands:

```
qlimaster export --format=both     # write quiz.csv / quiz.xlsx
qlimaster history rebuild          # rescan sibling quizzes into the team cache
qlimaster version
```

### Keys

| | |
|---|---|
| `e` enter-score · `i` edit · `a` add team | `:` config · `?` help · `q` quit |
| `hjkl` / arrows move · `g`/`G` top/bottom | `s` sort · `r` refresh · `R` read-out |
| `x` clear · `dd` delete team · `enter` confirm | `E` export (`c` csv · `x` xlsx · `b` both) |

## Files it writes

- **`quiz.hujson`** — the quiz state, in the current directory, saved on every change.
- **`quiz.csv` / `quiz.xlsx`** — exports, next to the quiz file.
- **`history.hujson`** — the team cache (names seen across quizzes, for fuzzy
  lookup). Located by walking up from the quiz directory to the nearest quiz
  root (a folder holding a `history.hujson` or dated quiz subfolders); if none
  is found it falls back to `$XDG_CONFIG_HOME/qlimaster/history.hujson`.

## Development

```
nix develop          # or: direnv allow
nix run .#test       # go test -race -cover ./...
nix run .#lint       # golangci-lint
nix build            # build the binary
nix run .#fuzz       # fuzz the score parser (nix run .#fuzz -- FuzzParse 60s)
```

`scripts/lint-ci.sh` / `scripts/lint-watch.sh` are `gh`-based CI-status helpers.

Commits run `golangci-lint` and the test suite via a [prek](https://github.com/j178/prek)
pre-commit hook (`prek install`).

## License

BSD-3-Clause. See [LICENSE](LICENSE).
