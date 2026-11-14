<?php

declare(strict_types=1);

/*
 * Copy of /plus (pages/plus.blade.php), from design/Web_Pricing.html. The FAQ is FAQ group `plus` (admin), not here.
 * Prices: the design's draft prices; `pricing.show` => false swaps every amount for «قیمت در اپ».
 * No discounts countdowns or scarcity copy (content red lines).
 */

return [
    'seo' => [
        'title' => 'ریتمی رایگان و ریتمی پلاس؛ چه چیزی همیشه رایگان است',
        'description' => 'پیش‌بینی پریود، ثبت روزانه، ابزارهای بارداری، یادآور واکسن و خروجی داده در ریتمی همیشه رایگان است. ریتمی پلاس فقط عمق و تحلیل بیشتر می‌دهد.',
    ],

    'intro' => [
        'eyebrow' => 'رایگان و پلاس',
        'title_line1' => 'ریتمی رایگان است.',
        'title_line2' => 'پلاس، عمق بیشتر.',
        'lead' => 'سلامت پایه‌ات هیچ‌وقت پشت پرداخت نمی‌رود. پلاس برای وقتی است که می‌خواهی بیشتر از داده‌ات بفهمی.',
        'trial' => 'همه پلن‌ها: ۷ روز رایگان، بدون کارت بانکی',
    ],

    // Single switch for the amounts: `show` false renders «قیمت در اپ» instead. Amounts are integers (toman);
    // `per_month` is the per-month price, `total` the whole period, `saving` the percent vs. monthly (null = none).
    'pricing' => [
        'show' => true,
        'in_app' => 'قیمت در اپ',
        'per_month' => 'تومان در ماه',
        'cancel_anytime' => 'هر وقت خواستی لغو کن',
        'saving' => ':percent٪ کمتر',
        'total_line' => ':total تومان، :tail',
        'plans' => [
            ['key' => 'monthly', 'title' => 'ماهانه', 'per_month' => 129000, 'total' => 129000, 'saving' => null, 'featured' => false],
            ['key' => 'quarterly', 'title' => '۳ ماهه', 'per_month' => 116000, 'total' => 348000, 'saving' => 10, 'featured' => false],
            ['key' => 'semiannual', 'title' => '۶ ماهه', 'per_month' => 99000, 'total' => 594000, 'saving' => 23, 'featured' => true],
            ['key' => 'yearly', 'title' => 'سالانه', 'per_month' => 82500, 'total' => 990000, 'saving' => 36, 'featured' => false],
        ],
    ],

    'cards' => [
        'label' => 'پلن‌های ریتمی پلاس',
        'brand' => 'ریتمی پلاس',
        'recommended' => 'پیشنهاد ما',
        'choose' => 'انتخاب',
        'why_six' => 'چرا ۶ ماهه؟ حدود شش چرخه داده، برای دیدن الگوی علائم و خوابت کافی است.',
    ],

    'includes' => [
        'title' => 'در همه پلن‌های پلاس، همه امکانات رایگان به‌علاوه:',
        'items' => [
            'همراه: همسر یا مادرت کنارت باشد',
            'ثبت با صدا',
            'تحلیل پیشرفته: الگوی علائم، خواب و حال',
            'یادآور پیامکی برای دارو و ویزیت',
            'گزارش روند چندماهه برای پزشک',
            'تفسیر دمای پایه و تست LH',
            'کتابخانه کامل مقاله‌ها و دوره‌ها',
        ],
    ],

    'free' => [
        'title' => 'نسخه رایگان، برای همیشه',
        'text' => 'پیگیری چرخه، ثبت علائم، یادآورهای داخل اپ، همه مرحله‌ها و هشدارهای سلامت.',
        'cta' => 'دانلود رایگان',
        'draft_note' => 'قیمت‌ها پیش‌نویس‌اند و بعد از بررسی نهایی اعلام می‌شوند.',
    ],

    'compare' => [
        'eyebrow' => 'مقایسه',
        'title' => 'دقیقاً چه چیزی در کدام است',
        'caption' => 'مقایسه نسخه رایگان و ریتمی پلاس',
        'feature' => 'قابلیت',
        'free' => 'رایگان',
        'plus' => 'پلاس',
        'yes' => 'دارد',
        'no' => 'ندارد',
        // [label, in free, in plus]
        'rows' => [
            ['پیش‌بینی پریود و باروری با سطح اطمینان', true, true],
            ['ثبت دستی علائم', true, true],
            ['یادآورهای داخل اپ', true, true],
            ['همه مرحله‌های زندگی', true, true],
            ['هشدارهای سلامت', true, true],
            ['خروجی و حذف داده', true, true],
            ['همراه (همدم)', false, true],
            ['ثبت با صدا', false, true],
            ['تحلیل پیشرفته', false, true],
            ['یادآور پیامکی', false, true],
            ['گزارش روند برای پزشک', false, true],
            ['تفسیر دما و LH', false, true],
        ],
    ],

    'why' => [
        'eyebrow' => 'چرا پلاس پولی است؟',
        'title' => 'بخش‌های اصلی ریتمی همیشه رایگان می‌ماند',
        'text' => 'هدف ما این است که آگاهی درباره سلامت زنان به همه برسد و این را مسئولیت خودمان می‌دانیم. اما نگه داشتن سرور، پردازش صدا، ارسال پیامک و نوشتن محتوای بازبینی‌شده هزینه دارد. پلاس کمک می‌کند این هزینه‌ها پوشش داده شود، بدون اینکه چیزی از نسخه رایگان کم شود.',
    ],

    'faq' => [
        'title' => 'سؤالات متداول',
    ],

    'app_cta' => [
        'title' => 'ریتمی را رایگان نصب کن',
        'lead' => 'بخش‌های اصلی همیشه رایگان. داده‌ات فقط مال خودت است.',
    ],
];
