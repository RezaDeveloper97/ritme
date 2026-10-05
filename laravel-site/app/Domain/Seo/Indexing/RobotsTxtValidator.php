<?php

declare(strict_types=1);

namespace App\Domain\Seo\Indexing;

use App\Support\Text\PersianDigits;

/**
 * Validates admin-edited robots.txt rules (RFC 9309 + the widely supported Crawl-delay / Clean-param) before they are
 * saved. Errors are Persian, with the line number. Guards against the classic production accident: a
 * `User-agent: *` group with `Disallow: /` (the whole site disappears from search). The `Sitemap:` line is appended
 * by RobotsTxt itself, so it is not accepted here.
 */
final class RobotsTxtValidator
{
    public const MAX_BYTES = 32_768;

    /**
     * @return list<string> empty when the rules are valid
     */
    public function errors(string $rules): array
    {
        $rules = str_replace("\r\n", "\n", trim($rules));
        if ($rules === '') {
            return [];
        }
        if (strlen($rules) > self::MAX_BYTES) {
            return ['حجم robots.txt نباید بیشتر از '.PersianDigits::number(self::MAX_BYTES / 1024).' کیلوبایت باشد.'];
        }
        if (! mb_check_encoding($rules, 'UTF-8')) {
            return ['robots.txt باید متن UTF-8 باشد.'];
        }

        $errors = [];
        $agents = [];          // user-agents of the current group
        $groupHasRules = false;
        $sawAgent = false;

        foreach (explode("\n", $rules) as $index => $raw) {
            $line = trim((string) preg_replace('/#.*$/u', '', $raw));
            $at = 'خط '.PersianDigits::toPersian($index + 1).': ';
            if ($line === '') {
                continue;
            }
            if (! str_contains($line, ':')) {
                $errors[] = $at.'قالب «نام: مقدار» رعایت نشده است.';

                continue;
            }

            [$field, $value] = array_map('trim', explode(':', $line, 2));
            $field = strtolower($field);

            switch ($field) {
                case 'user-agent':
                    if ($groupHasRules) { // a new group starts
                        $agents = [];
                        $groupHasRules = false;
                    }
                    if ($value === '' || preg_match('/^[A-Za-z0-9_\-.*\/ ]+$/', $value) !== 1) {
                        $errors[] = $at.'نام ربات (User-agent) نامعتبر است.';
                    }
                    $agents[] = strtolower($value);
                    $sawAgent = true;
                    break;

                case 'allow':
                case 'disallow':
                    $groupHasRules = true;
                    if ($agents === []) {
                        $errors[] = $at.'هر قانون باید بعد از یک خط User-agent بیاید.';
                    }
                    if ($value !== '' && (preg_match('/^[\/*]/', $value) !== 1 || preg_match('/\s/u', $value) === 1 || strlen($value) > 2048)) {
                        $errors[] = $at.'مسیر باید با «/» یا «*» شروع شود و فاصله نداشته باشد.';
                    }
                    if ($field === 'disallow' && in_array($value, ['/', '/*'], true) && in_array('*', $agents, true)) {
                        $errors[] = $at.'«Disallow: /» برای همه ربات‌ها کل سایت را از نتایج جست‌وجو حذف می‌کند و مجاز نیست.';
                    }
                    break;

                case 'crawl-delay':
                    $groupHasRules = true;
                    if ($agents === []) {
                        $errors[] = $at.'هر قانون باید بعد از یک خط User-agent بیاید.';
                    }
                    if (! is_numeric($value) || (float) $value < 0 || (float) $value > 60) {
                        $errors[] = $at.'Crawl-delay باید عددی بین ۰ تا ۶۰ باشد.';
                    }
                    break;

                case 'clean-param':
                    $groupHasRules = true;
                    if ($agents === [] || $value === '') {
                        $errors[] = $at.'Clean-param باید مقدار داشته باشد و بعد از User-agent بیاید.';
                    }
                    break;

                case 'sitemap':
                    $errors[] = $at.'خط Sitemap به‌طور خودکار اضافه می‌شود؛ آن را حذف کنید.';
                    break;

                default:
                    $errors[] = $at.'دستور ناشناخته «'.$field.'». دستورهای مجاز: User-agent، Allow، Disallow، Crawl-delay، Clean-param.';
            }
        }

        if (! $sawAgent) {
            $errors[] = 'دست‌کم یک خط User-agent لازم است.';
        }

        return $errors;
    }
}
