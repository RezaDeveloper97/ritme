<?php

declare(strict_types=1);

/*
 * Copy of /plus (pages/plus.blade.php), from design/html/plus.html. The FAQ «جواب سؤال‌هایی که حق داری بپرسی» is
 * FAQ group `plus` (admin, L3-09), not here.
 *
 * Prices (AUDIT §5.2 / open item 4 — the owner has not decided whether prices show on the site). The design only has
 * placeholders («[قیمت سالانه]», «[قیمت ماهانه]», «[قیمت هر بسته]»), never real or example values, so nothing is
 * invented: `pricing.show` is the single on/off switch, off by default, and every paid plan then reads «قیمت در اپ».
 * To publish prices set `show` => true and fill the amounts (integers, toman); an amount left null still falls back
 * to the in-app copy. No discounts, countdowns or scarcity copy (content red lines).
 * Design placeholders «[امکانات دیگر پلاس]» / «[بسته‌های دیگر]» are omitted until the owner lists them.
 */

return [
    'seo' => [
        'title' => 'ریتمی رایگان و ریتمی پلاس؛ چه چیزی همیشه رایگان است',
        'description' => 'پیش‌بینی پریود، ثبت روزانه، ابزارهای بارداری، یادآور واکسن و خروجی داده در ریتمی همیشه رایگان است. ریتمی پلاس و بسته‌های دوره فقط عمق و تحلیل بیشتر می‌دهند.',
    ],

    'intro' => [
        'eyebrow' => 'رایگان و پلاس',
        'title' => 'اول می‌گوییم چه چیزی همیشه رایگان است',
        'lead' => 'ریتمی ابزار را قفل نمی‌کند؛ اگر چیزی پولی است، عمق و تحلیل بیشتر است.',
    ],

    'pricing' => [
        'show' => false,
        'plus_yearly' => null,
        'plus_monthly' => null,
        'pack' => null,
        'in_app' => 'قیمت در اپ',
        'monthly' => 'ماهانه :price',
        'yearly_suffix' => 'در سال',
        'pack_suffix' => 'هر بسته',
    ],

    'plans' => [
        'label' => 'مقایسه نسخه رایگان، ریتمی پلاس و بسته‌های دوره',
        'free' => [
            'title' => 'همیشه رایگان',
            'price' => '۰ تومان',
            'note' => 'بدون محدودیت زمانی',
            'items' => [
                'پیش‌بینی پریود و باروری',
                'ثبت روزانه، دستی و با صدا',
                'ابزارهای بارداری: هفته، علائم، یادآور ویزیت، علائم خطر',
                'یادآور واکسن و نمودار رشد کودک',
                'حالت یائسگی و یادآور چکاپ',
                'همدم و خانواده',
                'خروجی کامل داده و حذف حساب',
            ],
        ],
        'plus' => [
            'title' => 'ریتمی پلاس',
            'note' => 'پرداخت سالانه یا ماهانه',
            'items' => [
                'تحلیل عمیق شخصی از داده خودت',
                'کتابخانه کامل محتوا و دوره‌ها',
                'گزارش‌های پیشرفته برای پزشک',
            ],
        ],
        'packs' => [
            'title' => 'بسته‌های دوره',
            'note' => 'پرداخت یک‌باره، بدون تعهد ادامه‌دار',
            'items' => [
                'آمادگی زایمان',
                'شیردهی',
            ],
        ],
    ],

    'faq' => [
        'eyebrow' => 'قبل از خرید',
    ],

    'app_cta' => [
        'title' => 'با نسخه رایگان شروع کن',
        'lead' => 'هر وقت خواستی پلاس را امتحان کن؛ یا هیچ‌وقت.',
    ],
];
