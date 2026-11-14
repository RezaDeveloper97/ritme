<?php

declare(strict_types=1);

namespace App\Domain\Content\Enums;

/**
 * Editorial photos of the marketing pages (home, stage heroes). The files are committed in
 * `resources/images/site/<value>.webp` (or `.jpg` for the older ones); Database\Seeders\SitePhotosSeeder imports them into the media library (so
 * they get the usual avif/webp variants through <x-picture>), and App\Domain\Content\SitePhotos finds them again by
 * content hash. Replacing a photo = replace the file and re-run the seeder.
 */
enum SitePhoto: string
{
    case HomeHero = 'home-hero-balcony';
    case CycleHero = 'cycle-hero-sofa-tea';
    case TtcHero = 'ttc-hero-park-bench';
    case PregnancyHero = 'preg-hero-window';
    case WindowTea = 'window-tea';
    case CoupleBreakfast = 'couple-breakfast';
    case PregnantWindow = 'pregnant-window';
    case MotherBaby = 'mother-baby-sleep';
    case MidlifePark = 'midlife-park';
    case MotherTeen = 'mother-teen-sofa';
    case PartnerTea = 'partner-tea';

    /** The committed file: WebP when present, else the older JPEG. */
    public function path(): string
    {
        $webp = resource_path('images/site/'.$this->value.'.webp');

        return is_file($webp) ? $webp : resource_path('images/site/'.$this->value.'.jpg');
    }

    public function alt(): string
    {
        return match ($this) {
            self::HomeHero => 'زنی خندان روی بالکن',
            self::CycleHero => 'زنی روی مبل با کیسه آب گرم و فنجان چای',
            self::TtcHero => 'زن و همسرش روی نیمکت در پارک',
            self::PregnancyHero, self::PregnantWindow => 'زن باردار کنار پنجره',
            self::WindowTea => 'زنی کنار پنجره با فنجان چای',
            self::CoupleBreakfast => 'زن و همسرش سر میز صبحانه',
            self::MotherBaby => 'مادر و نوزاد خوابیده',
            self::MidlifePark => 'زنی میانسال در پارک',
            self::MotherTeen => 'مادر و دختر نوجوان روی مبل',
            self::PartnerTea => 'مردی برای همسرش که روی مبل استراحت می‌کند چای می‌آورد',
        };
    }
}
