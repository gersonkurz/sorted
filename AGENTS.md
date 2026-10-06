# Repository Guidelines

## Project Structure & Module Organization

Sorted! is a Go reimplementation of the original esoteric language, faithful to the original including its quirks. Work is tracked as GitHub issues; the Go module is `github.com/gersonkurz/sorted`: `cmd/sorted` (CLI), `internal/numbers`, `internal/syntax`, `internal/interp`, `internal/emit` (legacy /D dump, and `--to-c`: C that behaves like the interpreter), `internal/render` (tables back to Sorted! text, `--lang`), `internal/cc` (C front end ported from chibicc), `internal/compile` (C to Sorted! tables, `--from-c`).

- `legacy/sorted.linux/` and `legacy/sorted.win32/`: original C++ implementations, platform build files, and sample `.s` programs.
- `manual/Sorted! - p-nand-q.com.html`: archived overview and examples.
- `examples/`: C programs and their Sorted! versions (`just examples` regenerates them), e.g. 99 Bottles of Beer.
- `CLAUDE.md`: porting requirements, architecture notes, and known legacy quirks.
- `justfile`: Go development and release workflows, expecting the executable package at `cmd/sorted/`.
- `out/`: ignored build, coverage, and release artifacts.

## Build, Test, and Development Commands

Install Go and `just`. Use the root `justfile`:

- `just build`: build the host executable as `out/build/sorted` (`.exe` on Windows).
- `just run legacy/sorted.win32/hello.s`: build and run a program (arguments pass through).
- `just test`: run all tests without cached results.
- `just test-one TestParseHello`: run tests matching a name or regex.
- `just test-race`: run tests with race detection (Unix only).
- `just coverage`: generate `out/coverage.out` and `out/coverage.html`.
- `just fmt`: format Go sources.
- `just verify`: format check, `go vet`, and all tests (the review loop's Verify step; works on Windows too).
- `just check`: `verify` plus Staticcheck; run before committing.
- `just package`: cross-compile release archives into `out/dist/`.

Packaging and the race detector require a Unix environment; everything else also runs under Windows `cmd`.

## Coding Style & Naming Conventions

Use `gofmt` formatting, including tabs for Go indentation. Follow standard Go naming: lowercase package names, `MixedCaps` identifiers, and exported names starting with capitals. Keep legacy reference files intact when implementing the port. Explain compatibility quirks in comments and tests.

## Testing Guidelines

Use Go’s standard `testing` package, colocated `*_test.go` files, and descriptive `TestXxx` names. Cover every implemented behavior, including parsing failures and legacy quirks; no numeric coverage threshold is configured.

Treat `legacy/*/*.s` as end-to-end golden-test inputs. Obtain expected output by running the original executable; ask the maintainer for captures when necessary. Do not infer golden output solely from C++ source. Preserve original semantics, output, and failure behavior.

## Commit & Pull Request Guidelines

History is limited to short descriptive subjects such as `Initial draft setup`; no formal commit convention is established. Use concise, action-oriented subjects and focused commits.

PRs should describe changed behavior, relevant compatibility decisions, and validation performed. Link related issues when applicable and identify any unavailable reference outputs or checks.
