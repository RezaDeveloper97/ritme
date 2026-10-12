-- 00093_insurance_policy_history.sql — insurance policy history and sealed questionnaire evidence (user decisions
-- 2026-10-12, CB-INS follow-up).
--
-- insurance_policies        a user keeps SEVERAL policies over time: «تمدید» creates a new policy (new period, members
--                           and coverages copied as a starting point) and the previous one becomes past / read-only
--                           with its claims. The one-per-user unique key goes; (user_id, valid_from) indexes the
--                           history (and keeps an index under the user_id foreign key). Exactly one policy is
--                           «current» (the latest period containing today, else the newest) — computed in Go, not
--                           stored.
-- insurance_questionnaires  +evidence_enc   the record items copied into the answers as evidence (titles of
--                           surgeries, conditions, medicines, …) sealed with PRIVATE_NOTE_KEY, bound to the owner
--                           (AD `insurance:questionnaire:<uid>`), as a JSON object {code: [items]}. `answers` keeps
--                           only code / answer / record_answer. Rows written before this migration keep their plain
--                           evidence in `answers` (still readable) until their next save, which seals it.
--
-- Health data, owner-only. Twin of backend/database/migrations/2026_10_12_000093_add_insurance_policy_history.php
-- (Laravel owns the prod schema until T-M2-27) so `make schema-diff` stays green; IF [NOT] EXISTS so it is a no-op
-- where the Laravel twin already ran.

-- +goose Up
ALTER TABLE `insurance_policies`
  ADD KEY IF NOT EXISTS `insurance_policies_user_id_valid_from_index` (`user_id`,`valid_from`);
ALTER TABLE `insurance_policies`
  DROP KEY IF EXISTS `insurance_policies_user_id_unique`;
ALTER TABLE `insurance_questionnaires` ADD COLUMN IF NOT EXISTS `evidence_enc` text DEFAULT NULL AFTER `answers`;

-- +goose Down
-- Restoring the one-per-user key fails while a user has more than one policy (the history must be removed first).
ALTER TABLE `insurance_questionnaires` DROP COLUMN IF EXISTS `evidence_enc`;
CREATE UNIQUE INDEX `insurance_policies_user_id_unique` ON `insurance_policies` (`user_id`);
DROP INDEX IF EXISTS `insurance_policies_user_id_valid_from_index` ON `insurance_policies`;
