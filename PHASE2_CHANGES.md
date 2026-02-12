# Phase 2 — Dependency Upgrades

## Chunk 1: Core runtime (Gin, sessions, cors, GORM)
Updated:
- `github.com/gin-gonic/gin` → `v1.11.0`
- `github.com/gin-contrib/sessions` → `v1.0.4`
- `github.com/gin-contrib/cors` → `v1.7.6`
- `gorm.io/gorm` → `v1.31.1`
- `gorm.io/driver/postgres` → `v1.6.0`
- `gorm.io/driver/sqlite` → `v1.6.0`

Validation: `go test ./...` ✅

## Chunk 2: Security + auth (Casbin, Go-Guardian, nosurf)
Updated:
- `github.com/casbin/casbin/v2` → `v2.135.0`
- `github.com/justinas/nosurf` → `v1.2.0`
- `github.com/shaj13/go-guardian/v2` remained on `v2.11.6` (latest resolvable in this module graph)

Validation: `go test ./...` ✅

## Chunk 3: Payments & email (stripe-go, mailjet, newrelic)
Updated:
- `github.com/mailjet/mailjet-apiv3-go/v4` → `v4.0.8`
- `github.com/stripe/stripe-go/v72` remained `v72.122.0` (newer target not resolvable in upstream tags)
- `github.com/newrelic/go-agent/v3` pinned to `v3.40.1`
- `github.com/newrelic/go-agent/v3/integrations/logcontext-v2/logWriter` pinned to `v1.0.2`
- `github.com/newrelic/go-agent/v3/integrations/nrgin` pinned to `v1.4.0`

Compatibility note: newer New Relic versions pulled a test import (`internal/integrationsupport`) that breaks tidy in this repo; pinned compatible versions keep tests green.

Validation: `go test ./...` ✅

## Chunk 4: Templ + frontend (templ, Tailwind)
Updated:
- `github.com/a-h/templ` → `v0.3.977`
- Tailwind: no `package.json` present in this repo, so no Node/Tailwind package change applied.

Validation: `go test ./...` ✅

## Final state
- `go mod tidy` executed
- `go mod vendor` executed
- Full test suite passes
