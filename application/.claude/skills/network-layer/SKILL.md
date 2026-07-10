---
name: network-layer
description: Use the hand-written Ritme HTTP stack (HttpClient + HttpResponseCache) correctly — no OkHttp/Retrofit, explicit-TTL GET caching, error mapping to AppResult, timeouts, transparent gzip. Use when adding any API call, endpoint adapter, or touching adapter/outbound/network, and to verify network code before finishing.
---

# Hand-Written Network Layer (no OkHttp/Retrofit)

All HTTP goes through `adapter/outbound/network/http/HttpClient` (CLAUDE.md §3).
Endpoint adapters (e.g. `AuthGatewayAdapter`) build `HttpRequest`s, parse with
`org.json`, and return `AppResult<T>` across the port — never a raw response and
never a throw.

## Rules
- **One client instance** — constructed only in `AppContainer`, base URL from `ApiConfig`.
- **Transport vs HTTP status**: a completed exchange (even 4xx/5xx) is
  `AppResult.Success(HttpResponse)`; the endpoint adapter decides how a non-2xx maps to
  `AppError.Http`. Only transport failures (timeout/DNS/IO) are `AppResult.Failure`.
- **Never set `Accept-Encoding` manually** — `HttpURLConnection` handles gzip
  transparently; setting it yourself disables the automatic decode (hook-enforced).
- **Never call `disconnect()`** — fully read + close streams so the socket returns to
  the keep-alive pool.
- **Timeouts always explicit** (client defaults: 10s connect / 15s read).

## GET response cache (§5)
`HttpResponseCache` = memory LRU (32 entries) + disk (`cacheDir/http_responses`),
per-entry absolute expiry, FNV-1a filenames. Opt-in per request:

```kotlin
HttpRequest(HttpMethod.GET, ApiConfig.BANNERS_PATH, cacheTtlMillis = 5 * 60_000L)
```

- Only 2xx GETs with `cacheTtlMillis > 0` are stored; errors and POSTs never cache.
- Pick TTLs by volatility: pregnancy week content (hours), banners (minutes),
  anything user-mutable (no cache, or invalidate with `clear()` after a write).
- Call `httpResponseCache.clear()` on logout — cached bodies may embed user data.

## Testing (§9)
Unit-test against a real local server — `com.sun.net.httpserver.HttpServer` ships in
the JDK (no new dependency). See `HttpClientTest` for the pattern: assert status/body
passthrough, header + query encoding, timeout → `AppError.Timeout`, unreachable →
`AppError.Network`, cache hit counts. Run:
`./gradlew --offline testDebugUnitTest`

## Checklist
- [ ] No OkHttp/Retrofit/raw connections outside `adapter/outbound/network|image` (hook: check_network.py).
- [ ] New GET-heavy endpoint sets a deliberate `cacheTtlMillis` (or documents why not).
- [ ] Endpoint adapter returns `AppResult`, maps non-2xx to `AppError.Http`.
- [ ] New endpoint has a `HttpClientTest`-style test if it exercises new client behavior.
