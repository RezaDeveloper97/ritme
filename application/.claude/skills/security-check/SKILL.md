---
name: security-check
description: Run a defensive security review of the Ritme app — secrets, TLS/network hardening, token storage, SQL injection, manifest/intent exposure, PII in logs/crash reports. Use before release, after touching auth/network/storage code, or when the user asks "is this secure."
---

# Security Check — Ritme

Run a security pass over the code in scope (a diff, a feature, or the whole `app/`). For a broad or whole-app pass, delegate to the `security-auditor` subagent; for a small diff, check inline.

## Checklist

1. **Secrets**: no hardcoded API keys, client secrets, tokens, or passwords in Kotlin/gradle/XML/JSON. Secrets come from `local.properties` → `BuildConfig`, or Android Keystore at runtime. Check `ritme-auth.json` / `ritmeApi.json` fixtures too.
2. **TLS/network**: only `https://` endpoints; no custom trust-all `TrustManager` or disabled hostname verification; tokens in headers, never query strings; timeouts set on the hand-written client.
3. **Token storage**: access/refresh tokens encrypted with an Android Keystore key before hitting SharedPreferences/SQLite — never plaintext at rest.
4. **SQLite**: parameterized `?` bindings everywhere; flag string-built SQL.
5. **Manifest/components**: no unnecessary `exported=true`; `allowBackup` deliberate; crash-recovery intent extras (`recovered_from_crash`, route) validated against a whitelist of routes before navigating.
6. **PII/logging**: breadcrumbs, `Log.*`, and crash payloads must not contain tokens, card numbers, national IDs (کد ملی), or phone numbers — mask them.
7. **Crypto**: only `javax.crypto` + Keystore; no home-rolled primitives (§3 exception process: ask the user).

## Output

Ranked findings (Critical → Low) with file:line, exploit scenario, and a fix that respects the zero-deps policy. State explicitly which categories passed clean.
