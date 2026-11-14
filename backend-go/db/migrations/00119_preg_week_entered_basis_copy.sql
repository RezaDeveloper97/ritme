-- 00119_preg_week_entered_basis_copy.sql — the fa body of preg.week_entered (goose 00113) glued a «ه» to the basis
-- placeholder («مبنای محاسبه هنوز {preg_basis}ه»), which reads broken once the label is filled. Replace it with the
-- inboxalerts/copy/fa.json wording. Only rows still holding the seeded sentence change, so an admin edit is left alone.
-- Data only; the preg.* rules exist only in the Go schema (00113 has no Laravel twin), so this has none either.

-- +goose Up
UPDATE `message_rules`
SET `content` = REPLACE(`content`, 'مبنای محاسبه هنوز {preg_basis}ه. بعد از سونوگرافی NT می‌تونی مبنا رو به‌روز کنی تا بازهٔ تخمین محدودتر بشه.', 'سن بارداری‌ت فعلاً بر اساس «{preg_basis}» حساب می‌شه. بعد از سونوگرافی NT می‌تونی مبنا رو به‌روز کنی تا تخمین دقیق‌تر بشه.')
WHERE `rule_key` = 'preg.week_entered';

-- +goose Down
UPDATE `message_rules`
SET `content` = REPLACE(`content`, 'سن بارداری‌ت فعلاً بر اساس «{preg_basis}» حساب می‌شه. بعد از سونوگرافی NT می‌تونی مبنا رو به‌روز کنی تا تخمین دقیق‌تر بشه.', 'مبنای محاسبه هنوز {preg_basis}ه. بعد از سونوگرافی NT می‌تونی مبنا رو به‌روز کنی تا بازهٔ تخمین محدودتر بشه.')
WHERE `rule_key` = 'preg.week_entered';
