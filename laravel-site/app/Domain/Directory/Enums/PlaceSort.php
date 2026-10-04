<?php

declare(strict_types=1);

namespace App\Domain\Directory\Enums;

/**
 * Listing order. There is no paid placement: «هیچ مجموعه‌ای برای بالاتر آمدن پول نمی‌دهد» (directory.html).
 * `Nearest` needs a reference point; without one SearchPlaces falls back to `Recommended`.
 */
enum PlaceSort: string
{
    case Recommended = 'recommended';
    case Nearest = 'nearest';
    case Rating = 'rating';
    case Price = 'price';
    case Newest = 'newest';

    public function label(): string
    {
        return match ($this) {
            self::Recommended => 'پیشنهادی',
            self::Nearest => 'نزدیک‌ترین',
            self::Rating => 'بهترین نظر',
            self::Price => 'ارزان‌ترین',
            self::Newest => 'جدیدترین',
        };
    }
}
