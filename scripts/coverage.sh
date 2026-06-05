#!/usr/bin/env bash
#
# Collect whole-module test coverage and write a merged profile to coverage.out.
#
# main() and the os.Exit shim in cli.Execute() only run in a real process, so
# `go test` alone can never count them. This script merges unit-test coverage
# with one run of the built binary (Go 1.20+ binary coverage), which covers
# those last statements and yields a true 100%.
#
# Outputs (both git-ignored):
#   coverage.out      merged profile, readable by `go tool cover`
#   coverage-pkg.txt  per-package percentages, for the CI summary
set -euo pipefail

cd "$(dirname "$0")/.."

covdir="$(mktemp -d)"
bindir="$(mktemp -d)"
trap 'rm -rf "$covdir" "$bindir"' EXIT
mkdir -p "$covdir/unit" "$covdir/bin"

# 1. Unit tests, emitting coverage in binary format (-coverpkg covers every
#    package so the binary run below can contribute to the same set).
go test -race -covermode=atomic -coverpkg=./... ./... \
  -args -test.gocoverdir="$covdir/unit"

# 2. Build an instrumented binary and exercise it once. `version` is enough to
#    reach main() and Execute(); the branch logic is covered by unit tests.
go build -cover -covermode=atomic -coverpkg=./... -o "$bindir/litmus" .
GOCOVERDIR="$covdir/bin" "$bindir/litmus" version >/dev/null

# 3. Merge both sources into a standard profile plus a per-package breakdown.
go tool covdata textfmt -i="$covdir/unit,$covdir/bin" -o=coverage.out
go tool covdata percent -i="$covdir/unit,$covdir/bin" >coverage-pkg.txt

go tool cover -func=coverage.out | tail -1
