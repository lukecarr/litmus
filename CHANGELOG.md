# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0](https://github.com/lukecarr/litmus/releases/tag/v0.3.0) - 2026-06-04

### Added

- `github` output format that emits inline annotations and a job summary for GitHub Actions (#19)
- GitHub Action for running Litmus in a workflow, with inputs that map to the CLI flags (#20)
- Cloudflare AI Gateway provider, selected with `--provider cloudflare` (#18)

### Changed

- Failed requests are now retried only on transient responses (5xx, 429, 408)

## [0.2.0](https://github.com/lukecarr/litmus/releases/tag/v0.2.0) - 2026-01-10

### Added

- HTML output format for test reports (#5)

### Fixed

- Version info now correctly populated when installed via `go install` (#6)

## [0.1.2](https://github.com/lukecarr/litmus/releases/tag/v0.1.2) - 2025-12-27

### Added

- Goreleaser for automated builds (#3)

## [0.1.1](https://github.com/lukecarr/litmus/releases/tag/v0.1.1) - 2025-12-27

### Added

- Version command and build info (#2)
- GitHub Actions release workflow (#2)

### Changed

- Updated model references and improved JSON Schema description in README (#1)
