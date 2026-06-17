# xianhu-chaos

`xianhu-chaos` is a local chaos mock platform for Xianhu third-party integrations.

V1 covers the Umember/Meituan coupon APIs used by `MtCouponUtils` and the Douyin coupon APIs used by `DyCouponUtils`. The project is structured so payment and IoT providers can be added as separate provider manifests later.

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

The console can set provider-wide scenarios, send common Umember/Douyin requests, inspect recent mock traffic, and reset runtime state.

Console logs are colorized by default. Set `NO_COLOR=1` before starting the server if plain logs are needed for file redirection or CI output.

## Connect xianhu-server

For local or test profile, point Umember to this service:

```yaml
app:
  umember:
    base-url: http://127.0.0.1:18080/
```

Keep the trailing `/`. `MtCouponUtils` builds URLs by concatenating `baseUrl + "open/login"`.

The default successful coupon detail uses `sku_id=1349572011`, which exists in the current test SQL for Meituan group-buying coupon plans.

Point Douyin to the same service:

```yaml
app:
  douyin:
    base-url: http://127.0.0.1:18080/
```

`DyCouponUtils` builds URLs from this base URL for token, prepare, and verify APIs.

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

Successful Douyin token:

```bash
curl -sS -X POST http://127.0.0.1:18080/oauth/client_token/ \
  -H 'Content-Type: application/json' \
  -d '{"client_key":"chaos-client-key","client_secret":"chaos-client-secret","grant_type":"client_credential"}'
```

Successful Douyin prepare:

```bash
curl -sS 'http://127.0.0.1:18080/goodlife/v1/fulfilment/certificate/prepare/?poi_id=7630290236999731263&code=102692315741346'
```

Successful Douyin verify:

```bash
curl -sS -X POST http://127.0.0.1:18080/goodlife/v1/fulfilment/certificate/verify/ \
  -H 'Content-Type: application/json' \
  -d '{"verify_token":"chaos-verify-token","poi_id":"7630290236999731263","encrypted_codes":["chaos-encrypted-code"]}'
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

## Douyin Scenarios

`configs/providers/douyin.yaml` is enabled by default and mocks:

- `POST /oauth/client_token/`
- `GET /goodlife/v1/fulfilment/certificate/prepare/`
- `POST /goodlife/v1/fulfilment/certificate/verify/`

Useful prepare coupon codes:

- `DY_PREPARE_500`: prepare returns HTTP 500
- `DY_PREPARE_FAIL`: prepare returns Douyin business failure
- `DY_PREPARE_BAD_JSON`: prepare returns malformed JSON
- `DY_EMPTY`: prepare returns no certificates
- `DY_MISSING_DATA`: prepare omits `data`
- `DY_UNKNOWN_SKU`: prepare returns an unknown `third_sku_id`
- `DY_VERIFY_FAIL`: prepare succeeds, then verify returns `券码已核销`
- `DY_VERIFY_500`: prepare succeeds, then verify returns HTTP 500
- `DY_VERIFY_BAD_JSON`: prepare succeeds, then verify returns malformed JSON

Useful verify tokens:

- `DY_VERIFY_FAIL_TOKEN`: verify returns `券码已核销`
- `DY_VERIFY_500_TOKEN`: verify returns HTTP 500
- `DY_VERIFY_BAD_JSON_TOKEN`: verify returns malformed JSON
