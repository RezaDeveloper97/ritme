# Secrets hygiene (T-M12-07, S-12 / S-13)

## S-12 — APP_KEY in the example env files
`.env.example` and `.env.stage.example` carried the same real-looking `APP_KEY` (the only value in git history). Both
are now empty with a comment on how to generate one; `scripts/secret-scan.sh` (rule `app_key_in_example`) keeps it out.
SHA-256 of the old value: `6289d60ad157abcbe3514320dbd9f194093c199fe496c495d68472009b5f0d69`. It decodes to 23 bytes,
not 32, so Laravel 12 (AES-256-CBC) cannot boot with it — a working server almost certainly has another key. Confirm
on the server (prints only MATCH / different / empty / missing, never a value):

```bash
OLD=6289d60ad157abcbe3514320dbd9f194093c199fe496c495d68472009b5f0d69; for f in /opt/ritme/.env /opt/ritme-stage/.env /opt/ritme-beta/.env /opt/ritme/backend/.env /opt/ritme-site/.env; do [ -r "$f" ] || { echo "$f: missing"; continue; }; v=$(grep -E '^APP_KEY=' "$f" | tail -1 | cut -d= -f2- | tr -d '\r' | sed -E 's/^["'\'']//; s/["'\'']$//'); if [ -z "$v" ]; then echo "$f: APP_KEY empty"; elif [ "$(printf %s "$v" | sha256sum | cut -c1-64)" = "$OLD" ]; then echo "$f: MATCH (old example key - rotate)"; else echo "$f: different"; fi; done; unset v
```

Running containers:

```bash
OLD=6289d60ad157abcbe3514320dbd9f194093c199fe496c495d68472009b5f0d69; for c in $(docker ps --format '{{.Names}}'); do v=$(docker exec "$c" printenv APP_KEY 2>/dev/null) || continue; [ -n "$v" ] || continue; [ "$(printf %s "$v" | sha256sum | cut -c1-64)" = "$OLD" ] && echo "$c: MATCH" || echo "$c: different"; done; unset v
```

Result: _(to fill after the user runs it)_.

### Rotation plan (only on MATCH, with the user)
1. `docker compose exec <laravel-app> php artisan key:generate --show` — write straight into `.env`, never print/log.
2. Move the old value to `APP_PREVIOUS_KEYS` (`backend/config/app.php` supports `previous_keys`), new one in `APP_KEY`.
3. Re-encrypt: today `backend/app` has no `encrypted` cast and no `Crypt::` — re-check with
   `grep -rnE "encrypted|Crypt::" backend/app`; if any column appears, a one-off artisan command reads (old key) and saves (new key).
4. Side effects: Blade admin sessions/cookies, CSRF and signed URLs reset (admins sign in again). Passport tokens are
   RSA-signed — app users stay signed in. backend-go does not use APP_KEY.
5. `config:clear`, restart app/worker/scheduler, smoke `/up`, admin login, one API call.
6. Clear `APP_PREVIOUS_KEYS` afterwards. Separate keys per env (prod, stage, beta, laravel-site).

## S-13 — committed contract Passport key
In `APP_ENV=production` backend-go refuses to boot when `STORAGE_PATH/oauth-public.key` has the fingerprint of the
committed contract key (`1a141d8e85c85ec28a98708906a14672422a8915244f9e342525a1f6d5ff7a08`;
`config.LoadFrom` → `passport.CheckStorageKey`). **Before the next beta deploy** (beta runs `APP_ENV=production`)
check its volume key is not the fixture — prints only public-key hashes:

```bash
for v in $(docker volume ls -q | grep -i storage); do f="$(docker volume inspect -f '{{.Mountpoint}}' "$v")/oauth-public.key"; [ -r "$f" ] && echo "$v: $(openssl pkey -pubin -in "$f" -outform DER | sha256sum | cut -c1-64)"; done
```

## Secret scan
`scripts/secret-scan.sh` (tracked files only, prints `file:line rule`, never the value): APP_KEY in `*.example`,
private-key headers outside `backend-go/contract/fixtures/keys`, AWS, GitHub, Google API, Slack tokens.
`--self-test` proves each rule. Gate: job `secrets` in `claude-to-stage.yml` (merge needs it).
