# Railway Deployment Notes (Render-Parity)

## Repository / Branch
- Repo: `/Users/art/.openclaw/workspace/armory`
- Branch: `modernization/go-1.26-v2`

## What was verified

### 1) Actual application entry point
- `find . -name "main.go" -type f | grep -v vendor` returns:
  - `./cmd/api/main.go`
  - `./internal/middleware/example/main.go`
- **Production server entry point is `cmd/api/main.go`**.
- `internal/middleware/example/main.go` is just an example app, not deployment entrypoint.

### 2) Tailwind/CSS requirement
- Base template references compiled CSS at:
  - `cmd/web/views/partials/base.templ` → `<link rel="stylesheet" href="/assets/css/output.css">`
- Static assets are embedded at compile time:
  - `cmd/web/efs.go` → `//go:embed "assets"`
- Static routes serve embedded assets:
  - `internal/server/static_routes.go` serves `/assets`
- **Conclusion:** Tailwind build output must be generated before Go build so `cmd/web/assets/css/output.css` is embedded into the binary.

### 3) Binary naming
- Render flow uses `app` in `make build`.
- Railway should also produce/run `app` for consistency.

## Corrected Railway approach

Use Dockerfile build and run the new non-interactive Make target that mirrors Render steps:
- install templ
- install tailwind binary
- generate templ files
- compile Tailwind CSS to `cmd/web/assets/css/output.css`
- build Go binary from `cmd/api/main.go` to `app`

### Dockerfile (current)
```dockerfile
# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder
RUN apk add --no-cache make curl
WORKDIR /app
COPY . .
RUN make build-railway

FROM alpine:3.22
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/app /app/app
COPY --from=builder /app/configs /app/configs
EXPOSE 8080
CMD ["/app/app"]
```

### railway.toml (current)
```toml
[build]
builder = "DOCKERFILE"
```

### Makefile target (current)
```make
build-railway:
	@echo "Generating templ files (pinned, non-interactive)..."
	@go run github.com/a-h/templ/cmd/templ@v0.3.977 generate ./...
	@echo "Installing tailwindcss binary (platform-aware, pinned)..."
	@OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	ARCH=$$(uname -m); \
	if [ "$$OS" = "darwin" ] && [ "$$ARCH" = "arm64" ]; then BIN="tailwindcss-macos-arm64"; \
	elif [ "$$OS" = "darwin" ] && [ "$$ARCH" = "x86_64" ]; then BIN="tailwindcss-macos-x64"; \
	elif [ "$$OS" = "linux" ] && [ "$$ARCH" = "x86_64" ]; then BIN="tailwindcss-linux-x64"; \
	elif [ "$$OS" = "linux" ] && [ "$$ARCH" = "aarch64" ]; then BIN="tailwindcss-linux-arm64"; \
	else echo "Unsupported platform: $$OS/$$ARCH"; exit 1; fi; \
	curl -sL "https://github.com/tailwindlabs/tailwindcss/releases/download/v3.4.10/$$BIN" -o tailwindcss && chmod +x tailwindcss
	@echo "Building CSS assets..."
	@./tailwindcss -i cmd/web/styles/input.css -o cmd/web/assets/css/output.css
	@echo "Building API with vendored modules..."
	@go build -mod=vendor -tags netgo -ldflags '-s -w' -o app cmd/api/main.go
```

## Local verification command
```bash
make build-railway
```

Expected output artifact:
- `./app`
