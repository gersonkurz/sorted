# Build, test & packaging tasks for sorted, the Go port of Sorted!.
#
# Cross-platform: macOS/Linux (sh) and Windows (cmd). Go commands are the same
# everywhere and live under "Shared"; the few recipes that need the shell are
# OS-gated with a native variant each, so `just --list` only shows what applies.
#
# Windows note: keep double quotes out of Windows recipe lines. They do not reliably
# survive the hand-off from just to cmd.exe (a quoted PowerShell -Command once
# arrived as a string literal and silently did nothing), and nothing here
# needs them.

# On Windows, run recipes through cmd. Ignored on macOS/Linux, which use sh.
set windows-shell := ["cmd.exe", "/c"]

# The freshly built binary, as the shell of this OS invokes it
bin := if os() == "windows" { "out\\build\\sorted.exe" } else { "out/build/sorted" }

# Version baked into the binary. Only the branch for this OS is evaluated.
# Release tags are the dialects (#39): cool, very-cool, very-very-cool, ...
# --version prints the dialect's sentence, and this string too when it is
# not exactly that tag.
version := if os() == "windows" { `git describe --tags --always --dirty 2>nul || echo dev` } else { `git describe --tags --always --dirty 2>/dev/null || echo dev` }

# Linker flags in a form that needs no quotes (see the Windows note above).
# Builds pass -buildvcs=false: the version comes from here, and VCS stamping
# makes go build fail outright where git refuses the repository (e.g. a
# network share owned by another user, as with the Windows VM).
ldflags := "-ldflags=-X=main.version=" + version

# The examples directory, as the shell of this OS spells paths
ex := if os() == "windows" { "examples\\" } else { "examples/" }

# Targets for `just package` and `just cross`, as GOOS/GOARCH pairs. The tool
# is pure Go, so any target costs one line; nobody runs most of these, but Go
# cross-compiles reliably, and `just cross` checks that each one builds.
# linux/s390x: Sorted! on an IBM mainframe. plan9/amd64: an esoteric language
# on an esoteric OS.
platforms := "darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 linux/386 linux/arm linux/riscv64 linux/ppc64le linux/s390x windows/amd64 windows/386 windows/arm64 freebsd/amd64 freebsd/arm64 openbsd/amd64 netbsd/amd64 illumos/amd64 plan9/amd64 aix/ppc64"

# Default recipe: show available commands
default:
    @just --list

# ============================================================================
# Shared
# ============================================================================

# Build the sorted interpreter for this platform into out/build
build:
    go build -buildvcs=false -trimpath {{ldflags}} -o {{bin}} ./cmd/sorted

# Build, then run the interpreter with arguments passed through
run *args: build
    {{bin}} {{args}}

# Run all tests (never replayed from the test cache)
test:
    go test -count 1 ./...

# Format all Go sources
fmt:
    gofmt -w .

# Run go vet
vet:
    go vet ./...

# Run staticcheck (pinned as a tool in go.mod)
staticcheck:
    go tool staticcheck ./...

# Format check, go vet and all tests: the review loop's Verify step
verify: fmt-check vet test

# Format check, go vet and staticcheck
lint: fmt-check vet staticcheck

# Lint, test, and build for every release platform: run before committing
check: lint test cross

# Check that the tool builds for every platform `just package` ships
cross $SORTED_CROSS="1":
    go test -count 1 -run TestCrossCompile ./cmd/sorted

# Regenerate the Sorted! versions (English and German) of the C examples
examples:
    go run ./cmd/sorted --from-c {{ex}}99-bottles.c --lang en > {{ex}}99-bottles.s
    go run ./cmd/sorted --from-c {{ex}}99-bottles.c --lang de > {{ex}}99-bottles.de.s
    go run ./cmd/sorted --from-c {{ex}}brainfuck.c --lang en > {{ex}}brainfuck.s
    go run ./cmd/sorted --from-c {{ex}}brainfuck.c --lang de > {{ex}}brainfuck.de.s

# Run tests with coverage; writes out/coverage.out and out/coverage.html
coverage: _out-dir
    go test -count 1 -coverprofile=out/coverage.out ./...
    go tool cover -html=out/coverage.out -o out/coverage.html

# ============================================================================
# macOS / Linux
# ============================================================================

# gofmt -l exits 0 when it merely lists files, and non-zero on a syntax error
# (possibly with nothing listed), so both need checking.
#
# Fail if any Go source needs gofmt
[unix]
fmt-check:
    #!/usr/bin/env sh
    unformatted=$(gofmt -l .) || exit 1
    if [ -n "$unformatted" ]; then
        echo "Files need gofmt:"
        echo "$unformatted"
        exit 1
    fi

# The pattern reaches go test through the environment ($pattern), so regex
# characters such as | are never parsed by the shell.
#
# Run tests whose name matches a regex, verbosely: just test-one 'TestA|TestB'
[unix]
test-one $pattern:
    go test -count 1 -v -run "$pattern" ./...

# Run all tests with the race detector (needs cgo, hence not on Windows)
[unix]
test-race:
    go test -count 1 -race ./...

# One archive per platform with the binary, README, LICENSE, the legacy
# sample programs and the examples, plus SHA256SUMS.
#
# Cross-compile all platforms into out/dist
[unix]
package: clean-dist
    #!/usr/bin/env sh
    set -e
    mkdir -p out/dist
    for platform in {{platforms}}; do
        goos=${platform%/*}
        goarch=${platform#*/}
        name="sorted-{{version}}-${goos}-${goarch}"
        stage="out/stage/${name}"
        exe=""
        [ "$goos" = "windows" ] && exe=".exe"
        echo "Building ${name}..."
        mkdir -p "${stage}/examples"
        CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -buildvcs=false -trimpath {{ldflags}} -o "${stage}/sorted${exe}" ./cmd/sorted
        cp README.md LICENSE "${stage}/"
        cp legacy/sorted.win32/*.s examples/*.c examples/*.s examples/*.in "${stage}/examples/"
        if [ "$goos" = "windows" ]; then
            (cd out/stage && zip -qr "../dist/${name}.zip" "${name}")
        else
            tar -czf "out/dist/${name}.tar.gz" -C out/stage "${name}"
        fi
    done
    rm -rf out/stage
    (cd out/dist && shasum -a 256 * > SHA256SUMS)
    echo "Packages written to out/dist:"
    ls -1 out/dist

# Remove release archives
[unix]
clean-dist:
    rm -rf out/dist out/stage

# Remove all build output
[unix]
clean:
    rm -rf out

[unix]
[private]
_out-dir:
    @mkdir -p out

# ============================================================================
# Windows (cmd)
# ============================================================================

# A gofmt syntax error fails the first line; findstr succeeds only if gofmt
# listed a file.
#
# Fail if any Go source needs gofmt
[windows]
fmt-check: _out-dir
    gofmt -l . > out\gofmt.txt
    @findstr . out\gofmt.txt && (echo Files need gofmt & exit 1) || exit 0

# The pattern reaches go test through the environment and is expanded by a
# nested cmd with delayed expansion (!pattern!), which happens after cmd has
# parsed | & < >, so regex characters stay literal without any quoting.
#
# Run tests whose name matches a regex, verbosely: just test-one "TestA|TestB"
[windows]
test-one $pattern:
    cmd /v:on /c go test -count 1 -v -run !pattern! ./...

# Remove all build output
[windows]
clean:
    if exist out rmdir /s /q out

[windows]
[private]
_out-dir:
    @if not exist out mkdir out
