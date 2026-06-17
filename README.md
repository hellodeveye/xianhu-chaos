# xianhu-chaos

`xianhu-chaos` is a local chaos mock platform for Xianhu third-party integrations.

V1 focuses on the Umember/Meituan coupon APIs used by `MtCouponUtils`. The project is structured so Douyin, payment, and IoT providers can be added as separate provider manifests later.

## Run

```bash
cd /Users/kim/projects/xianhu/xianhu-chaos
go run ./cmd/server
```

Default address:

```text
http://127.0.0.1:18080
```

Health check:

```bash
curl -sS http://127.0.0.1:18080/health
```

Web console:

```text
http://127.0.0.1:18080/
```

The console can set provider-wide scenarios, send common Umember requests, inspect recent mock traffic, and reset runtime state.

## Connect xianhu-server

For local or test profile, point Umember to this service:

```yaml
app:
  umember:
    base-url: http://127.0.0.1:18080/
```

Keep the trailing `/`. `MtCouponUtils` builds URLs by concatenating `baseUrl + "open/login"`.

The default successful coupon detail uses `sku_id=1349572011`, which exists in the current test SQL for Meituan group-buying coupon plans.

## Provider Manifests

Provider manifests live in:

```text
configs/providers/*.yaml
```

The files use a JSON-compatible YAML subset so the service can run without external parser dependencies. A provider declares:

- `routes`: HTTP method, path, default scenario, and optional coupon-code field.
- `scenarios`: HTTP status, fixture path, content type, and chaos options.
- `rules`: coupon-code or header matches that map requests to scenarios.

Startup validation fails fast when:

- a fixture is missing
- a route references an unknown scenario
- a rule references an unknown scenario
- two enabled providers register the same `method + path`

## Chaos Selection Order

The selected scenario is resolved in this order:

1. Request header `X-Chaos-Scenario`
2. Coupon-code rules in the provider manifest
3. Provider global scenario set by admin API
4. Route default scenario

## Common Examples

Successful login:

```bash
curl -sS -X POST http://127.0.0.1:18080/open/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","password":"test"}'
```

Successful Meituan detail:

```bash
curl -sS 'http://127.0.0.1:18080/open/user/meituan/coupon/detail?store_id=8674228&coupon_code=0109760017002' \
  -H 'Authorization: chaos-mock-token'
```

Trigger detail HTTP 500 by coupon code:

```bash
curl -i 'http://127.0.0.1:18080/open/user/meituan/coupon/detail?store_id=8674228&coupon_code=DETAIL_500'
```

Trigger verify failure by coupon code:

```bash
curl -sS -X POST http://127.0.0.1:18080/open/user/meituan/coupon/verify \
  -H 'Content-Type: application/json' \
  -H 'Authorization: chaos-mock-token' \
  -d '{"store_id":"8674228","coupon_code":"VERIFY_FALSE"}'
```

Force one request with a header:

```bash
curl -i 'http://127.0.0.1:18080/open/user/meituan/coupon/detail?store_id=8674228&coupon_code=0109760017002' \
  -H 'X-Chaos-Scenario: detail_bad_json'
```

Set a provider-wide global scenario:

```bash
curl -sS -X PUT http://127.0.0.1:18080/__admin/providers/umember/scenario \
  -H 'Content-Type: application/json' \
  -d '{"scenario":"login_http_500"}'
```

Reset state:

```bash
curl -sS -X POST http://127.0.0.1:18080/__admin/reset
```

Inspect state:

```bash
curl -sS http://127.0.0.1:18080/__admin/state
```

## Douyin Status

`configs/providers/douyin.yaml` is included as a disabled template with fixtures for:

- `POST /oauth/client_token/`
- `GET /goodlife/v1/fulfilment/certificate/prepare/`
- `POST /goodlife/v1/fulfilment/certificate/verify/`

`DyCouponUtils` currently hardcodes `https://open.douyin.com/...`, so xianhu-server needs a later Java-side base URL configuration before it can use this provider directly.
