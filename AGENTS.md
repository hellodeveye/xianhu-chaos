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
- `internal/chaos` — `Engine` does scenario selection, delay/jitter/errorRate/failOnNth, and holds all in-memory state (route scenarios, shared global scenarios, request counts, recent request log, scenario response overrides). Lost on restart; `POST /__admin/reset` clears it.
- `internal/httpserver` — routing. Admin/UI routes use Go 1.22+ method patterns (`GET /health`, `GET /{$}`); provider routes use path-only `HandleFunc` with in-handler method dispatch returning 405 on miss. Follow this split when adding routes.
- `internal/admin` — `/__admin/*` + `/health`. `PUT /__admin/providers/{name}/scenario` sets a shared global scenario; empty scenario clears it. `PUT /__admin/providers/{name}/routes/{routeId}/scenario` sets a route-owned scenario; empty scenario clears it. `GET/PUT/DELETE /__admin/providers/{name}/scenarios/{scenario}` view/override/restore a scenario's response (body + status + contentType, in-memory). `PUT /__admin/providers/` prefix coexists with the more-specific `{name}/scenarios/{scenario}` patterns — Go 1.22 ServeMux resolves the latter first; covered by `TestScenarioOverrideRoutePrecedence`.
- `internal/ui` — `//go:embed static` from `internal/ui/static/`. Rebuild after editing static files.

## Provider manifests

`configs/providers/*.yaml` (JSON). Every scenario must explicitly declare either `routeId` or `scope: "shared"`; there is no name-prefix ownership fallback. Startup fails fast when: a fixture is missing, a route references an unknown scenario, a scenario omits ownership, a scenario references an unknown routeId, a route default is not owned by that route, a rule references an unknown scenario, two enabled providers register the same `method + path`, or a provider name duplicates. Same path with different methods is allowed.

Fixture paths in manifests are **project-root-relative** (e.g. `fixtures/umember/login_success.json`), resolved via `../../<fixture>` from the manifest dir. Add new fixtures under `fixtures/<provider>/` and reference from root.

## Tests depend on real manifests + fixtures

`internal/provider` and `internal/httpserver` tests load the real `configs/providers/*.yaml` (via relative `../../configs/providers`) and exercise real fixtures. Editing any manifest/fixture can break tests in unrelated packages — keep scenario/route/rule references consistent. Test packages run from their own dir, hence the `../../` paths.

## Scenario selection order

Resolved in `chaos.Engine.Select`: `X-Chaos-Scenario` header (`chaos.ScenarioHeader`) → coupon-code/header rules → route scenario (admin API) → provider shared global scenario (admin API) → route default. Header and rule scenarios apply only when owned by the current route or marked `scope: "shared"`; incompatible route-owned scenarios are ignored. After the scenario is chosen, any in-memory response override (set via admin API/UI) is applied on top of the fixture — override wins over fixture content but not over scenario selection. Runtime route/global/response overrides are cleared by `POST /__admin/reset`; response overrides can also be cleared by `DELETE /__admin/providers/{name}/scenarios/{scenario}`. See `README.md` for the catalog of coupon-code triggers and curl examples.
