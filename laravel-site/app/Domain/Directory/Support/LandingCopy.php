<?php

declare(strict_types=1);

namespace App\Domain\Directory\Support;

use App\Domain\Directory\Data\CategoryData;
use App\Domain\Directory\Data\CityData;
use App\Domain\Directory\Data\LandingData;

/**
 * Copy of a directory landing page (`/directory/{city}`, `/directory/{city}/{category}`): the admin-edited row
 * (directory_landings) where filled, otherwise these templates. Red lines: no superlatives or pressure.
 */
final class LandingCopy
{
    public static function resolve(CityData $city, ?CategoryData $category, ?LandingData $edited = null): LandingData
    {
        $fallback = $category === null ? self::forCity($city) : self::forCategory($city, $category);

        return new LandingData(
            cityId: $city->id,
            categoryId: $category?->id,
            h1: self::filled($edited?->h1) ?? $fallback->h1,
            title: self::filled($edited?->title) ?? $fallback->title,
            description: self::filled($edited?->description) ?? $fallback->description,
            intro: self::filled($edited?->intro) ?? $fallback->intro,
        );
    }

    private static function forCity(CityData $city): LandingData
    {
        return new LandingData(
            cityId: $city->id,
            categoryId: null,
            h1: "خدمات مادر و کودک در {$city->name}",
            title: "خدمات مادر و کودک در {$city->name}",
            description: "کلاس مادر و کودک، استخر، خانه بازی و کارگاه‌های {$city->name} را ببین، صفحه هر مجموعه را بخوان و وقت رزرو کن.",
            intro: null,
        );
    }

    private static function forCategory(CityData $city, CategoryData $category): LandingData
    {
        return new LandingData(
            cityId: $city->id,
            categoryId: $category->id,
            h1: "{$category->name} در {$city->name}",
            title: "{$category->name} در {$city->name}",
            description: "{$category->name} در {$city->name}: ساعت کاری، رده سنی، خدمات و قیمت‌ها و نظر مادرها را ببین و وقت رزرو کن.",
            intro: null,
        );
    }

    private static function filled(?string $value): ?string
    {
        return $value === null || trim($value) === '' ? null : trim($value);
    }
}
