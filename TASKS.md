# Modbus CLI Tasks

## Phase 1 - Scaffold

- [x] Add Go module, Makefile, thin entrypoint, and repo layout.
- [x] Add config, output, CLI, and Modbus wrapper packages.

## Phase 2 - Config and Wrapper

- [x] Implement YAML defaults, profiles, validation, starter config, and CLI override support.
- [x] Hide `github.com/grid-x/modbus` behind `internal/modbusclient`.

## Phase 3 - Commands

- [x] Implement `init-config`, `validate-config`, `test-connection`/`status`, `read`, `write`, `identify`, `watch`, `completions`, and `version`.
- [x] Keep writes dry-run by default and require `--yes` for execution.

## Phase 4 - Output and Docs

- [x] Implement table, text, JSON, JSON Lines, and CSV rendering.
- [x] Replace the placeholder README with first-run, config, command, safety, output, troubleshooting, completions, and exit-code documentation.

## Phase 5 - Verification

- [x] Run `go test ./...`.
- [x] Run `make build`.
- [x] Smoke-test help, config generation, config validation, and completions.

## Phase 6 - Device Maps

- [x] Add generic YAML named point definitions.
- [x] Add point validation, profile merging, and duplicate-name checks.
- [x] Add `points`, `read-point`, `write-point`, and `watch-point`.
- [x] Apply scale/offset on reads and reverse them before writes.
- [x] Keep SunSpec support isolated behind a future package boundary without importing the incompatible `gosunspec/modbus` adapter.
