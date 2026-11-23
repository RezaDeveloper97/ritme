---
id: T-M12-07
title: Secrets hygiene: blank APP_KEY in examples, refuse the contract key in production, CI secret grep
milestone: M12
type: backend
status: done
depends_on: []
parallel_group: M12-SEC
touches: [docs/quality-2026-11/secrets.md, backend-go/internal/platform/config/passport_key_test.go, .env.example, .env.stage.example, backend-go/internal/platform/config/config.go, backend-go/internal/auth/passport, .github/workflows/claude-to-stage.yml, scripts/secret-scan.sh]
skills: []
verify: bash scripts/secret-scan.sh && cd backend-go && CGO_ENABLED=0 go test ./internal/platform/config/... ./internal/auth/passport/...
---

# T-M12-07 — Secrets hygiene: blank APP_KEY in examples, refuse the contract key in production, CI secret grep
**Model:** opus. Evidence: `docs/quality-2026-11/findings.md`.

## Why
S-12/S-13: a real-looking Laravel APP_KEY is committed in two example files; the committed contract Passport key could be loaded in prod by mistake.

## Scope
1. Replace `APP_KEY` in both examples with an empty value and a comment on how to generate one.
2. Ask the user to compare the server `.env` APP_KEY with the old example value themselves (give the exact grep; never print either value). If they match, plan the rotation with the user (encrypted casts must be re-encrypted) — do not rotate unilaterally.
3. Go: at startup in `APP_ENV=production`, refuse to boot if the loaded Passport public key fingerprint equals the contract fixture key.
4. `scripts/secret-scan.sh`: fails on `APP_KEY=base64:.{40,}` in `*.example`, private-key headers outside `contract/fixtures`, and common token patterns; run it in CI.

## Out of scope
- Rotating real keys without the user.
- Android (queue rule). Production deploys (queue rule).

## Acceptance
- Scan green on the tree; unit test for the fingerprint guard; the user has been told the result of the server check.
- Test cases reviewed: every new test has at least one negative/boundary case and fails when the fix is reverted (say how you checked).
