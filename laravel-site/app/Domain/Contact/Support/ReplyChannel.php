<?php

declare(strict_types=1);

namespace App\Domain\Contact\Support;

use App\Support\Text\PersianDigits;

/**
 * The form's single «ایمیل یا شماره همراه» field, split into the channel it is: an email address (RFC syntax only —
 * no DNS lookup, that would be an external request) or an Iranian mobile number, normalised to `09xxxxxxxxx`
 * (Persian/Arabic digits, spaces, dashes, `+98` / `0098` prefixes accepted). Anything else is not a reply channel.
 */
final readonly class ReplyChannel
{
    private function __construct(public ?string $email, public ?string $phone) {}

    public static function parse(?string $value): ?self
    {
        $value = trim((string) $value);
        if ($value === '') {
            return null;
        }

        if (str_contains($value, '@')) {
            $email = mb_strtolower($value);

            return mb_strlen($email) <= 191 && filter_var($email, FILTER_VALIDATE_EMAIL) !== false ? new self($email, null) : null;
        }

        $digits = preg_replace('/[\s\-().]/u', '', PersianDigits::toLatin($value)) ?? '';
        if (preg_match('/^(?:\+98|0098|98|0)?(9\d{9})$/', $digits, $m) !== 1) {
            return null;
        }

        return new self(null, '0'.$m[1]);
    }
}
