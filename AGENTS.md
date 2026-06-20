# AGENTS.md

Local chaos mock platform for Xianhu third-party integrations (Umember/Meituan, Douyin). Go stdlib only — `go.mod` has zero external dependencies by design. Do not add a YAML parser, HTTP framework, or router library without strong justification.

## Commands

Run from repo root (config/providers/fixtures paths are relative to root):

- Run server: `go run ./cmd/server` or `task run` (default `:18080`, config `configs/config.yaml`). Override with `task run ADDR=:19090 CONFIG=path`.
- Pre-commit gate: `task check` = `fmt` → `vet` → `test`. Run this before considering work done.
- Single package: `go test ./internal/httpserver`. Single test: `go test ./internal/httpserver -run TestHeaderScenarioTakesPrecedence`.
- `task fmt` runs `gofmt -w ./cmd ./internal` only — it does **not** touch `configs/` or `fixtures/`. Format those JSON files manually (2-space indent). Do not run `gofmt -w .` expecting full coverage.
- `NO_COLOR=1` disables ANSI colors in console logs (use for CI/file redirection).

## Critical: config files are JSON, not YAML

`configs/config.yaml` and `configs/providers/*.yaml` carry a `.yaml` extension but are parsed by `encoding/json` (`internal/config/config.go`, `internal/provider/loader.go`). They must be valid JSON: double-quoted keys, no comments, no multi-doc, no bare strings. The README calls this a "JSON-compatible YAML subset"; the executable truth is JSON-only. Writing real YAML will fail startup and tests.

## Architecture

Entry: `cmd/server/main.go` (-config flag) → `config.Load` → `provider.LoadRegistry` → `chaos.Engine` → `httpserver.Server` wires `ui` + `admin` + provider routes.

- `internal/provider` — manifest loading + validation; `Registry.Routes` keyed by path, method matched in handler.
- `internal/chaos` — `Engine` does scenario selection, delay/jitter/errorRate/failOnNth, and holds all in-memory state (global scenarios, request counts, recent request log, scenario overrides). Lost on restart; `POST /__admin/reset` clears it.
- `internal/httpserver` — routing. Admin/UI routes use Go 1.22+ method patterns (`GET /health`, `GET /{$}`); provider routes use path-only `HandleFunc` with in-handler method dispatch returning 405 on miss. Follow this split when adding routes.
- `internal/admin` — `/__admin/*` + `/health`. `PUT /__admin/providers/{name}/scenario` sets global scenario; empty scenario clears it. `GET/PUT/DELETE /__admin/providers/{name}/scenarios/{scenario}` view/override/restore a scenario's response (body + status + contentType, in-memory). `PUT /__admin/providers/` prefix coexists with the more-specific `{name}/scenarios/{scenario}` patterns — Go 1.22 ServeMux resolves the latter first; covered by `TestScenarioOverrideRoutePrecedence`.
- `internal/ui` — `//go:embed static` from `internal/ui/static/`. Rebuild after editing static files.

## Provider manifests

`configs/providers/*.yaml` (JSON). Startup fails fast when: a fixture is missing, a route references an unknown scenario, a rule references an unknown scenario, two enabled providers register the same `method + path`, or a provider name duplicates. Same path with different methods is allowed.

Fixture paths in manifests are **project-root-relative** (e.g. `fixtures/umember/login_success.json`), resolved via `../../<fixture>` from the manifest dir. Add new fixtures under `fixtures/<provider>/` and reference from root.

## Tests depend on real manifests + fixtures

`internal/provider` and `internal/httpserver` tests load the real `configs/providers/*.yaml` (via relative `../../configs/providers`) and exercise real fixtures. Editing any manifest/fixture can break tests in unrelated packages — keep scenario/route/rule references consistent. Test packages run from their own dir, hence the `../../` paths.

## Scenario selection order

Resolved in `chaos.Engine.Select`: `X-Chaos-Scenario` header (`chaos.ScenarioHeader`) → coupon-code/header rules → provider global scenario (admin API) → route default. The header must name an existing scenario in that provider or it is silently ignored. After the scenario is chosen, any in-memory override (set via admin API/UI) is applied on top of the fixture — override wins over fixture content but not over scenario selection. Overrides are cleared by `POST /__admin/reset` or `DELETE /__admin/providers/{name}/scenarios/{scenario}`. See `README.md` for the catalog of coupon-code triggers and curl examples.
