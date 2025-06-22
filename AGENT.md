# Mox Email Server - Agent Development Guide

## Build/Test/Lint Commands
- **Build**: `make build` (includes frontend compilation)
- **Test**: `make test` (standard tests), `make test-race` (race detection), `make test-integration` (full integration)
- **Single test**: `CGO_ENABLED=0 go test -run TestName ./package` 
- **Lint**: `make check` (requires `make install-staticcheck install-ineffassign`)
- **Format**: `make fmt` (Go + gofmt)
- **Frontend**: `make frontend` (TypeScript compilation)

## Architecture
- **Core**: Go 1.23+ mail server with SMTP, IMAP, webmail, admin interfaces
- **Components**: ~40 packages (smtp, imap, dkim, spf, dmarc, queue, store, etc.)
- **Database**: bstore (bbolt-based) for message metadata, file storage for messages
- **Frontend**: TypeScript/vanilla JS, no frameworks
- **APIs**: Sherpa-based JSON APIs, webhooks for events
- **Metrics**: Prometheus-compatible metrics throughout

## Code Style & Conventions
- Follow existing Go conventions, no special formatter rules
- Use `mlog` package for logging (wraps slog), `tcheck(t, err, msg)` for test errors
- Imports: standard lib first, external, then mox packages with `github.com/mjl-/mox/`
- Error handling: panics used extensively, `xerr` helpers for context
- Testing: standard Go testing, build tags for integration (`//go:build !integration`)
- No internal/ directory by design, reusable packages avoid mox-specific dependencies
