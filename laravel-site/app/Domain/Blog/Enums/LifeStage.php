<?php

declare(strict_types=1);

namespace App\Domain\Blog\Enums;

/**
 * The six life stages of the site (stage pages, stage colours `stage-*` in @theme). A post or category may belong
 * to one; stage pages list «برای همین مرحله» readings by it.
 */
enum LifeStage: string
{
    case Cycle = 'cycle';
    case Ttc = 'ttc';
    case Pregnancy = 'pregnancy';
    case Postpartum = 'postpartum';
    case Menopause = 'menopause';
    case Teen = 'teen';

    public function label(): string
    {
        return match ($this) {
            self::Cycle => 'چرخه و پریود',
            self::Ttc => 'اقدام به بارداری',
            self::Pregnancy => 'بارداری',
            self::Postpartum => 'پس از زایمان و کودک',
            self::Menopause => 'یائسگی',
            self::Teen => 'نوجوان و والدین',
        };
    }
}
