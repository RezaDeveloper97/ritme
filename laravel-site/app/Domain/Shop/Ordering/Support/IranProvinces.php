<?php

declare(strict_types=1);

namespace App\Domain\Shop\Ordering\Support;

/**
 * The 31 provinces of Iran with their larger cities — local data for the checkout address (no external service).
 * The province is stored by key; the city is free text (small towns are not listed) suggested from these lists.
 */
final class IranProvinces
{
    /** @var array<string, array{name: string, cities: list<string>}> */
    private const PROVINCES = [
        'east-azerbaijan' => ['name' => 'آذربایجان شرقی', 'cities' => ['تبریز', 'مراغه', 'مرند', 'میانه', 'اهر', 'بناب', 'سهند']],
        'west-azerbaijan' => ['name' => 'آذربایجان غربی', 'cities' => ['ارومیه', 'خوی', 'بوکان', 'مهاباد', 'میاندوآب', 'سلماس', 'پیرانشهر']],
        'ardabil' => ['name' => 'اردبیل', 'cities' => ['اردبیل', 'پارس‌آباد', 'مشگین‌شهر', 'خلخال', 'گرمی']],
        'isfahan' => ['name' => 'اصفهان', 'cities' => ['اصفهان', 'کاشان', 'خمینی‌شهر', 'نجف‌آباد', 'شاهین‌شهر', 'شهرضا', 'فولادشهر', 'مبارکه']],
        'alborz' => ['name' => 'البرز', 'cities' => ['کرج', 'فردیس', 'نظرآباد', 'هشتگرد', 'محمدشهر', 'ماهدشت']],
        'ilam' => ['name' => 'ایلام', 'cities' => ['ایلام', 'دهلران', 'ایوان', 'آبدانان']],
        'bushehr' => ['name' => 'بوشهر', 'cities' => ['بوشهر', 'برازجان', 'بندر گناوه', 'کنگان', 'جم']],
        'tehran' => ['name' => 'تهران', 'cities' => ['تهران', 'اسلامشهر', 'شهریار', 'قدس', 'ملارد', 'پاکدشت', 'ورامین', 'رباط‌کریم', 'دماوند', 'پردیس']],
        'chaharmahal-bakhtiari' => ['name' => 'چهارمحال و بختیاری', 'cities' => ['شهرکرد', 'بروجن', 'فرخ‌شهر', 'لردگان']],
        'south-khorasan' => ['name' => 'خراسان جنوبی', 'cities' => ['بیرجند', 'قائن', 'طبس', 'فردوس']],
        'razavi-khorasan' => ['name' => 'خراسان رضوی', 'cities' => ['مشهد', 'نیشابور', 'سبزوار', 'تربت حیدریه', 'قوچان', 'کاشمر', 'تربت جام']],
        'north-khorasan' => ['name' => 'خراسان شمالی', 'cities' => ['بجنورد', 'شیروان', 'اسفراین', 'آشخانه']],
        'khuzestan' => ['name' => 'خوزستان', 'cities' => ['اهواز', 'دزفول', 'آبادان', 'بندر ماهشهر', 'اندیمشک', 'خرمشهر', 'بهبهان', 'شوشتر']],
        'zanjan' => ['name' => 'زنجان', 'cities' => ['زنجان', 'ابهر', 'خرمدره', 'قیدار']],
        'semnan' => ['name' => 'سمنان', 'cities' => ['سمنان', 'شاهرود', 'دامغان', 'گرمسار']],
        'sistan-baluchestan' => ['name' => 'سیستان و بلوچستان', 'cities' => ['زاهدان', 'زابل', 'چابهار', 'ایرانشهر', 'خاش', 'سراوان']],
        'fars' => ['name' => 'فارس', 'cities' => ['شیراز', 'مرودشت', 'جهرم', 'فسا', 'کازرون', 'صدرا', 'داراب', 'لار']],
        'qazvin' => ['name' => 'قزوین', 'cities' => ['قزوین', 'تاکستان', 'الوند', 'بوئین‌زهرا']],
        'qom' => ['name' => 'قم', 'cities' => ['قم']],
        'kurdistan' => ['name' => 'کردستان', 'cities' => ['سنندج', 'سقز', 'مریوان', 'بانه', 'قروه']],
        'kerman' => ['name' => 'کرمان', 'cities' => ['کرمان', 'سیرجان', 'رفسنجان', 'جیرفت', 'بم', 'زرند']],
        'kermanshah' => ['name' => 'کرمانشاه', 'cities' => ['کرمانشاه', 'اسلام‌آباد غرب', 'کنگاور', 'هرسین', 'سنقر']],
        'kohgiluyeh-boyer-ahmad' => ['name' => 'کهگیلویه و بویراحمد', 'cities' => ['یاسوج', 'دوگنبدان', 'دهدشت']],
        'golestan' => ['name' => 'گلستان', 'cities' => ['گرگان', 'گنبد کاووس', 'علی‌آباد کتول', 'بندر ترکمن', 'آزادشهر']],
        'gilan' => ['name' => 'گیلان', 'cities' => ['رشت', 'بندر انزلی', 'لاهیجان', 'لنگرود', 'تالش', 'آستارا', 'صومعه‌سرا']],
        'lorestan' => ['name' => 'لرستان', 'cities' => ['خرم‌آباد', 'بروجرد', 'دورود', 'الیگودرز', 'کوهدشت']],
        'mazandaran' => ['name' => 'مازندران', 'cities' => ['ساری', 'بابل', 'آمل', 'قائم‌شهر', 'بهشهر', 'چالوس', 'نوشهر', 'تنکابن']],
        'markazi' => ['name' => 'مرکزی', 'cities' => ['اراک', 'ساوه', 'خمین', 'محلات', 'دلیجان']],
        'hormozgan' => ['name' => 'هرمزگان', 'cities' => ['بندرعباس', 'میناب', 'قشم', 'کیش', 'بندر لنگه']],
        'hamadan' => ['name' => 'همدان', 'cities' => ['همدان', 'ملایر', 'نهاوند', 'تویسرکان', 'اسدآباد']],
        'yazd' => ['name' => 'یزد', 'cities' => ['یزد', 'میبد', 'اردکان', 'بافق', 'مهریز']],
    ];

    /**
     * key ⇒ Persian name, in Persian alphabetical order (the constant is kept sorted by hand: no intl on cPanel).
     *
     * @return array<string, string>
     */
    public static function options(): array
    {
        return array_map(static fn (array $p): string => $p['name'], self::PROVINCES);
    }

    public static function has(string $key): bool
    {
        return isset(self::PROVINCES[$key]);
    }

    public static function name(string $key): string
    {
        return self::PROVINCES[$key]['name'] ?? $key;
    }

    /**
     * Every listed city (for the city suggestions), unique, grouped by province.
     *
     * @return list<string>
     */
    public static function cities(): array
    {
        return array_values(array_unique(array_merge(...array_values(array_map(static fn (array $p): array => $p['cities'], self::PROVINCES)))));
    }
}
