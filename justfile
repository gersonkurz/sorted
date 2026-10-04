# Set executable suffix based on operating system
EXE_SUFFIX := if os() == "windows" { ".exe" } else { "" }

# Output directory for native builds
BUILD_DIR := if os() == "macos" { "out/build/mac" } else if os() == "windows" { "out\\build\\win-64" } else { "out/build/linux" }

# Output directory for release archives
DIST_DIR := "out/dist"

set windows-shell := ["cmd", "/C"]

# Auto-computed build metadata
version := `git describe --tags --always --dirty 2>/dev/null || echo dev`
LDFLAGS := "-s -w -X main.version=" + version

# Targets for `just package`, as GOOS/GOARCH pairs
PLATFORMS := "darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 linux/386 windows/amd64 windows/386"

# Default target
default: build

# Build the sorted interpreter for the host platform
build:
    go build -buildvcs=false -trimpath -ldflags "{{LDFLAGS}}" -o {{BUILD_DIR}}/sorted{{EXE_SUFFIX}} ./cmd/sorted
    @echo Built {{BUILD_DIR}}/sorted{{EXE_SUFFIX}}

# Usage: just run <arguments for sorted>
# Build, then run the interpreter with arguments passed through
run *ARGS: build
    {{BUILD_DIR}}/sorted{{EXE_SUFFIX}} {{ARGS}}

# Run all tests
test:
    go test -count 1 ./...

# Usage: just test-one TestParseHello
# Run tests whose name matches a regex, verbosely
test-one PATTERN:
    go test -count 1 -v -run '{{PATTERN}}' ./...

# Run all tests with the race detector
test-race:
    go test -count 1 -race ./...

# Run tests with coverage; writes out/coverage.out and an HTML report
[unix]
coverage:
    mkdir -p out
    go test -count 1 -coverprofile=out/coverage.out ./...
    go tool cover -func=out/coverage.out | tail -1
    go tool cover -html=out/coverage.out -o out/coverage.html
    @echo Coverage report: out/coverage.html

# Format all Go sources
fmt:
    gofmt -w .

# Check formatting, run go vet and staticcheck
[unix]
lint:
    #!/usr/bin/env sh
    set -e
    unformatted=$(gofmt -l .)
    if [ -n "$unformatted" ]; then
        echo "Files need gofmt:"
        echo "$unformatted"
        exit 1
    fi
    go vet ./...
    go run honnef.co/go/tools/cmd/staticcheck@latest ./...

# Format check, lint and test: run before committing
check: lint test

# Cross-compile all PLATFORMS and package each into out/dist with README, LICENSE and the sample programs
[unix]
package: clean-dist
    #!/usr/bin/env sh
    set -e
    mkdir -p {{DIST_DIR}}
    for platform in {{PLATFORMS}}; do
        goos=${platform%/*}
        goarch=${platform#*/}
        name="sorted-{{version}}-${goos}-${goarch}"
        stage="out/stage/${name}"
        exe=""
        [ "$goos" = "windows" ] && exe=".exe"
        echo "Building ${name}..."
        mkdir -p "${stage}/examples"
        CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -buildvcs=false -trimpath -ldflags "{{LDFLAGS}}" -o "${stage}/sorted${exe}" ./cmd/sorted
        cp README.md LICENSE "${stage}/"
        cp legacy/sorted.linux/*.s "${stage}/examples/"
        if [ "$goos" = "windows" ]; then
            (cd out/stage && zip -qr "../../{{DIST_DIR}}/${name}.zip" "${name}")
        else
            tar -czf "{{DIST_DIR}}/${name}.tar.gz" -C out/stage "${name}"
        fi
    done
    rm -rf out/stage
    (cd {{DIST_DIR}} && shasum -a 256 * > SHA256SUMS)
    echo "Packages written to {{DIST_DIR}}:"
    ls -1 {{DIST_DIR}}

# Remove release archives
[unix]
clean-dist:
    rm -rf {{DIST_DIR}} out/stage

# Clean all build artifacts
[unix]
clean:
    rm -rf out

[windows]
clean:
    if exist out rmdir /s /q out
