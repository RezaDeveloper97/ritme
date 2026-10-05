<?php

declare(strict_types=1);

namespace App\Domain\Seo\Audit;

/**
 * How bad an audit finding is. Errors fail `seo:audit` (CI); warnings fail it only with `--strict`; notices are hints
 * that never fail a run (e.g. a slow-ish response, an indexable page missing from the sitemap).
 */
enum Severity: string
{
    case Error = 'error';
    case Warning = 'warning';
    case Notice = 'notice';

    public function label(): string
    {
        return match ($this) {
            self::Error => 'خطا',
            self::Warning => 'هشدار',
            self::Notice => 'نکته',
        };
    }

    public function color(): string
    {
        return match ($this) {
            self::Error => 'danger',
            self::Warning => 'warning',
            self::Notice => 'gray',
        };
    }

    /** Points a page loses per finding (page score = 100 − penalties, floor 0). */
    public function penalty(): int
    {
        return match ($this) {
            self::Error => 20,
            self::Warning => 5,
            self::Notice => 1,
        };
    }

    /**
     * @return array<string, string> value => Persian label, for selects
     */
    public static function options(): array
    {
        $options = [];
        foreach (self::cases() as $case) {
            $options[$case->value] = $case->label();
        }

        return $options;
    }
}
