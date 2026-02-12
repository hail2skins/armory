# Build Troubleshooting Report

## Context
- Repository: `/Users/art/.openclaw/workspace/armory`
- Branch: `modernization/go-1.26-v2`
- Go: `go1.26.0 darwin/arm64`

## Commands Run & Findings

### 1) Reproduce build issue
Command:
```bash
go build -ldflags=-w -s -o out ./cmd/api
```
Actual error:
```text
flag provided but not defined: -s
usage: go build [-o output] [build flags] [packages]
Run 'go help build' for details.
```

### 2) Correctly quoted ldflags build
Command:
```bash
go build -v -ldflags='-w -s' -o out ./cmd/api
```
Result:
- Succeeds (`github.com/hail2skins/armory/cmd/api` built)

### 3) Go environment
Commands:
```bash
go version
go env GOROOT GOPATH CGO_ENABLED
which go
```
Output:
- `go version go1.26.0 darwin/arm64`
- `GOROOT=/Users/art/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.0.darwin-arm64`
- `GOPATH=/Users/art/go`
- `CGO_ENABLED=1`
- `which go -> /opt/homebrew/bin/go`

### 4) Module checks
Commands:
```bash
head -20 go.mod
go mod verify
```
Result:
- `go.mod` uses `go 1.26.0`
- `go mod verify` => `all modules verified`

### 5) Vendor checks
Commands:
```bash
ls -la vendor/
go mod vendor
```
Result:
- `vendor/` exists and is valid after sync
- `vendor/modules.txt` present

### 6) Entry point checks
Commands:
```bash
ls -la cmd/api/
ls -la cmd/web/
```
Result:
- Both `cmd/api` and `cmd/web` exist

### 7) Alternative build attempts
Commands:
```bash
go build -o out ./cmd/web
go build -v ./cmd/web
```
Result:
- Build path is valid (no source/entrypoint issue)

## Root Cause
The failing command used **incorrect ldflags syntax**:
- ❌ `-ldflags=-w -s`

`-s` was parsed as a top-level `go build` flag instead of part of linker flags, causing:
- `flag provided but not defined: -s`

## Fix Applied
Use quoted linker flags so both flags are passed under `-ldflags`:
- ✅ `go build -ldflags='-w -s' -o out ./cmd/api`

Equivalent valid form:
- `go build -ldflags "-w -s" -o out ./cmd/api`

## Notes
- No code changes were required.
- The issue was command invocation, not module layout, vendor integrity, CGO, or missing entrypoints.
