# qlimaster

Keyboard-driven, full-screen terminal UI for hosting pub quizzes.

Built for running the pub quiz at Grandcafe de Burcht in Leiden, as a
replacement for a spreadsheet.

## Status

Early development.

## Features (planned)

- Spreadsheet-style live score table that uses the full terminal window.
- Fast score entry: round number, fuzzy team picker, score, repeat.
- Fuzzy team lookup against previous quizzes for quick setup.
- Automatic sorting (worst to best) after every round.
- Perfect-round highlighting.
- Halftime and final cumulative score columns (configurable).
- Per-column score averages.
- Always-saved HuJSON state; CSV and XLSX export on demand.

## Quick start

```
cd ~/quiz/$(date +%Y-%m-%d)
qlimaster
```

Override defaults with flags:

```
qlimaster --rounds=12 --questions=24 --checkpoints=4,8,12
```

Subcommands:

```
qlimaster export --format=both
qlimaster history rebuild
qlimaster version
```

## Development

Enter the dev shell (requires Nix with flakes):

```
nix develop
```

Or with direnv + nix-direnv:

```
direnv allow
```

Run tests, lint, build:

```
./scripts/test.sh
./scripts/lint.sh
./scripts/build.sh
```

Check CI lint status via `gh`:

```
./scripts/lint-ci.sh
./scripts/lint-watch.sh
```

## License

BSD-3-Clause. See LICENSE.
