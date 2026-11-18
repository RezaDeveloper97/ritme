# User tables — what account deletion removes (T-M12-13, finding T-03)

`DELETE /api/v1/account` (`backend-go/internal/profile/service.go` → `Service.DeleteAccount`) runs, in one
transaction: delete the user's refresh + access tokens, the companion pregnancy notices she sent, the OTP rows of her
number, then the `users` row — every other user row goes by `ON DELETE CASCADE`. After the commit it removes her
support screenshots and her encrypted files of every purpose (`files.RemoveUser`) and her per-user Redis keys
(`profile.PurgeUserKeys`).

Two tests in `backend-go/internal/profile/delete_complete_int_test.go` keep this true:

- **Schema test** — every column named `user_id`, `owner_id`, `member_id`, `child_owner_id` or `*_user_id` has an
  `ON DELETE CASCADE` foreign key to `users`, **or** is in the allow-list below (the test parses this table). A new
  table without either fails; an allow-list row whose column is gone or now cascades fails too (stale entry).
- **Behaviour test** — one row in every user table (every table with a user column, and every table below them
  through foreign keys) for the deleted user, a control user, and "link" rows that join her to other users; one
  stored file each; Redis keys each; then `DELETE /api/v1/account`: no column that points at users holds her id, every
  row of hers that should go is gone, the control user's rows are all there, files and keys are gone, her token is
  dead, and an OTP login with the same number creates a new, empty account.

## Allow-list

Kinds: `deleted-by-code` (no FK, removed by name in `DeleteAccount`) · `same-row` (another column of the same row
cascades) · `anonymised` (FK `ON DELETE SET NULL`: the row stays without her id) · `kept` (not her data or not
reachable, stays) · `pending-decision` (may have legal retention — stays until the user decides).

| Column | Kind | Reason | Deleted by |
|---|---|---|---|
| `oauth_access_tokens.user_id` | deleted-by-code | Passport table (Laravel-owned until T-M2-27); a big table, no FK added | `DeleteUserAccessTokens` (+ `DeleteUserRefreshTokens` for `oauth_refresh_tokens`) in `DeleteAccount` |
| `ivf_cycles.active_user_id` | same-row | unique helper = `user_id` while the cycle is open | `ivf_cycles.user_id` CASCADE |
| `insurance_claims.member_id` | same-row | points at `insurance_policy_members`, not `users` (SET NULL) | `insurance_claims.user_id` CASCADE |
| `ai_usage_logs.user_id` | anonymised | provider cost log: feature, model, tokens, cost — no content | FK SET NULL |
| `city_service_listings.owner_user_id` | anonymised | a venue's public listing outlives the account that claimed it | FK SET NULL |
| `city_service_reports.user_id` | anonymised | a moderation report about a venue (reason + note) stays for the venue's record | FK SET NULL |
| `sessions.user_id` | kept | Laravel web sessions; under the `admin` guard it holds an `admins.id`, so no FK is possible; app users never get a web session (API tokens only) | — (Laravel session GC) |
| `oauth_clients.owner_id` | kept | polymorphic (`owner_type`); Ritme's clients are system clients without an owner | — |
| `plus_admin_actions.user_id` | pending-decision | Plus billing ledger (amounts, gateway refs) written by admins | — kept until the user decides (see below) |

## Leaking before this task (fixed by goose `00129_user_fk_cascades.sql`)

Rows that stayed after an account delete (no FK, no hand delete):

| Table | What stayed | Fix |
|---|---|---|
| `message_events` | every smart-message impression / click / goal of the user | FK CASCADE (orphans deleted first) |
| `admin_health_access_requests` | admin requests to see her health data (reason, ticket) | FK CASCADE |
| `lab_jobs` | lab queue rows (only went if the lab report went first) | FK CASCADE |
| `oauth_auth_codes` | Passport auth codes (no Ritme flow writes them today) | FK CASCADE |
| `oauth_device_codes` | Passport device codes (idem) | FK CASCADE |
| `user_profiles_dedupe_backup` | her duplicate profile rows saved by 00096 (birthday, weight, LMP…) | FK CASCADE |
| `otp_verifications` | OTP rows of her number (keyed by mobile, no id) | `DeleteOtpsForMobile` in `DeleteAccount` |
| Redis | cycle-engine views (health data, 24 h), idempotent responses (24 h), message / activity marks, family read-audit marks | `PurgeUserKeys` after the commit |

The migration deletes the orphan rows of those six tables (rows of accounts deleted earlier — nobody else's data),
then adds the FKs with `foreign_key_checks = 0` (`ALGORITHM=INPLACE, LOCK=NONE`: no table copy on `message_events`).

## Family and companion links

Her link rows go, the other side's data stays: `companions` (`owner_id` and `companion_user_id`), `families`
(`owner_id`, `spouse_user_id`), `family_*` (`owner_id`, `member_id`), `family_share_requests` (`from_user_id`,
`to_user_id`), `family_proposals` (`actor_id`, `approver_id`), `child_guardians.user_id` cascade. Rows she wrote in a
shared space owned by someone else (`children`, `child_*`, `baby_*`, `todo_*` `created_by` / `updated_by`,
`todo_tasks.assigned_to`, `child_guardians.invited_by`, `companion_audit_logs.actor_id`, `companion_invites.used_by_id`)
are `SET NULL`: the row stays with the owner, without her id. Plain ids without an FK that may still name her in
someone else's audit trail: `companion_audit_logs.undone_by`, `family_proposals.decided_by` (a number only).

## Not deleted on purpose

- `learning_sms_suppressions.phone` — an SMS opt-out must survive account deletion.
- `companion_invites.phone` of invites **another** user sent to her number — the inviter's data.
- `admin_audit_logs.target_id` — append-only admin audit (ids and scalar flags only).
- Redis: OTP guard keys (`otp:*:<sha256(mobile)>`, no id or number; keeping them stops a delete-and-resignup from
  resetting the send / failure budget, they expire on their own), rate-limit counters (`throttle:*`, short TTL,
  counts only), queued jobs (their handlers find no user).

## Pending user decision — payment / ledger retention

These hold receipts, invoices or money movements that may have a legal retention period. **Not changed by this
task**; the question is open:

| Table | Today |
|---|---|
| `plus_admin_actions` | kept (no FK) |
| `plus_invoices`, `plus_receipts`, `plus_store_purchases`, `plus_subscriptions`, `plus_trials` | deleted (CASCADE) |
| `shop_orders` (+ `shop_order_items`, `shop_order_events`, `shop_order_shipments`), `shop_returns` | deleted (CASCADE) |
| `ai_credit_grants`, `ai_credit_spends` | deleted (CASCADE) |
| `telemed_bookings`, `city_service_bookings`, `insurance_claims` | deleted (CASCADE) |

Question: should any of them be kept (anonymised: `user_id` → NULL, buyer PII cleared) instead of deleted?

## Other delete paths

The admin delete (`internal/admin/users.Destroy`) shares the cascades but not the hand deletes: it **revokes** her
access tokens instead of deleting them (`oauth_access_tokens` / `oauth_refresh_tokens` rows stay), keeps the OTP rows
and the Redis keys. Open follow-up (outside T-M12-13's paths).
