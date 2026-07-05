<?php

namespace App\Services\HomePage\Support;

use App\Enums\CyclePhase;

/**
 * Maps a cycle phase to a display color + label, so the week-calendar and other
 * sections present phases consistently. Colors echo the app's pink/coral theme.
 */
final class CyclePhasePalette
{
    private const COLORS = [
        'menstruation' => '#F2567C', // قاعدگی
        'follicular' => '#7BC99A',   // فولیکولار
        'ovulation' => '#F09BB5',    // تخمک‌گذاری
        'luteal' => '#9C8BD6',       // لوتئال
    ];

    public static function color(?string $phase): ?string
    {
        if ($phase === null) {
            return null;
        }

        return self::COLORS[$phase] ?? '#CFCFE6';
    }

    public static function label(?string $phase, string $locale = 'fa'): ?string
    {
        if ($phase === null) {
            return null;
        }

        $enum = CyclePhase::tryFrom($phase);

        return $enum?->label($locale);
    }

    /**
     * @return array{phase: ?string, phase_label: ?string, color: ?string}
     */
    public static function describe(?string $phase, string $locale = 'fa'): array
    {
        return [
            'phase' => $phase,
            'phase_label' => self::label($phase, $locale),
            'color' => self::color($phase),
        ];
    }
}
