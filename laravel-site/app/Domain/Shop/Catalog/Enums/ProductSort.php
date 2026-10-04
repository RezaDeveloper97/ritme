<?php

declare(strict_types=1);

namespace App\Domain\Shop\Catalog\Enums;

/**
 * Listing sorts of the design (shop-list «ترتیب»). Values are the `?sort=` query values. Out-of-stock products always
 * go last. No paid placement.
 */
enum ProductSort: string
{
    case BestSelling = 'best-selling';
    case Newest = 'newest';
    case Cheapest = 'cheapest';
    case TopRated = 'top-rated';

    public static function default(): self
    {
        return self::BestSelling;
    }

    public function label(): string
    {
        return match ($this) {
            self::BestSelling => 'پرفروش‌ترین',
            self::Newest => 'جدیدترین',
            self::Cheapest => 'ارزان‌ترین',
            self::TopRated => 'بیشترین امتیاز',
        };
    }
}
