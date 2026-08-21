<?php

namespace App\Enums;

/**
 * Writing direction of a language, stored on every languages row and mirrored
 * into the frontend's <html dir>. It is a property of the locale, not a guess
 * derived from its code — an admin picks it when creating the language.
 */
enum TextDirection: string
{
    case RTL = 'rtl';
    case LTR = 'ltr';

    public function label(string $locale = 'en'): string
    {
        return match ($locale) {
            'fa' => match ($this) {
                self::RTL => 'راست‌به‌چپ (RTL)',
                self::LTR => 'چپ‌به‌راست (LTR)',
            },
            default => match ($this) {
                self::RTL => 'Right to left (RTL)',
                self::LTR => 'Left to right (LTR)',
            },
        };
    }

    public static function values(): array
    {
        return array_column(self::cases(), 'value');
    }
}
