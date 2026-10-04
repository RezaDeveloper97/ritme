<?php

declare(strict_types=1);

/*
 * /postpartum — design/html/postpartum.html. Structure (icons, colours, links, mock screens) lives in
 * App\Domain\Content\Stages\Postpartum; the keys of `features`, `tools`, `help` and `hero.float` match its specs.
 * `mock` is the copy of the decorative phone screen mock/screens/postpartum-baby (demo values, aria-hidden); the
 * family split uses the shared companion screen (stages/common.mock.companion).
 * Red lines: no diagnosis claims, no «حتماً/قطعاً/دقیق‌ترین/تضمینی», no sales pressure.
 */

return [
    'name' => 'پس از زایمان و کودک',

    'seo' => [
        'title' => 'پس از زایمان: بهبودی مادر، شیردهی و رشد کودک',
        'description' => 'بهبودی بعد از زایمان، شیردهی و خواب را ثبت کن و رشد، واکسن و نقاط عطف کودک را دنبال کن؛ با ثبت مشترک همدم و پروفایل جدا برای هر فرزند.',
    ],

    'hero' => [
        'eyebrow' => 'پس از زایمان و کودک',
        // h1 = title with the lilac `highlight` phrase in place of :highlight (design wording).
        'highlight' => 'حال خودت',
        'title' => ':highlight هم مهم است، نه فقط کودک',
        'lead' => 'بهبودی بعد از زایمان، شیردهی و خواب، و رشد، واکسن و نقاط عطف کودک؛ برای یک یا چند فرزند.',
        'float' => [
            'vaccine' => ['title' => 'واکسن ۴ ماهگی', 'text' => '۳ روز دیگر'],
            'weight' => ['title' => 'وزن آوا', 'text' => '۶٫۸ کیلو · صدک ۵۰'],
        ],
    ],

    'features' => [
        'recovery' => [
            'eyebrow' => 'مادر',
            'title' => 'بهبودی تو، قدم‌به‌قدم',
            'text' => 'خونریزی، خواب و حال روزهای بعد از زایمان را ثبت کن؛ اگر چیزی ارزش پیگیری داشت، آرام خبرت می‌کنیم.',
            'points' => ['پرسش کوتاه حال هفتگی', 'یادآور ویزیت پس از زایمان', 'راهنمای «کی فوراً تماس بگیرم»'],
        ],
        'baby' => [
            'eyebrow' => 'کودک',
            'title' => 'رشد، واکسن و نقاط عطف',
            'screen_title' => 'رشد',
            'text' => 'نمودار رشد، یادآور واکسن‌ها و نقاط عطف رشدی؛ دفتر ثبت و یادآور، نه تشخیص بیماری کودک.',
            'points' => ['یادآور واکسن', 'نمودار قد و وزن', 'ثبت شیر، خواب و پوشک'],
        ],
        'family' => [
            'eyebrow' => 'خانواده',
            'title' => 'ثبت مشترک با همدم',
            'text' => 'همسرت هم می‌تواند شیر و خواب کودک را ثبت کند تا کارها تقسیم شود.',
            'points' => ['ثبت از دو گوشی', 'پروفایل جدا برای هر فرزند', 'دسترسی قابل کنترل'],
        ],
    ],

    'tools' => [
        'safe-home' => ['title' => 'چک‌لیست خانه امن', 'text' => 'اتاق‌به‌اتاق، برای ماه‌های اول'],
        'buying-guide' => ['title' => 'راهنمای انتخاب', 'text' => 'صندلی ماشین، کالسکه و شیردوش'],
        'vaccines' => ['title' => 'یادآور واکسن', 'text' => 'از بدو تولد'],
    ],

    'help' => [
        'directory' => ['title' => 'خدمات مادر و کودک', 'text' => 'کلاس، استخر و خانه بازی نزدیک تو، با رزرو.'],
        'clinicians' => ['title' => 'پزشک و ماما', 'text' => 'ویزیت آنلاین یا حضوری با پرونده‌ای که خودت اجازه‌اش را می‌دهی.'],
        'shop' => ['title' => 'فروشگاه', 'text' => 'سیسمونی و محصولات بهداشتی؛ جدا از داده سلامت.'],
    ],

    'emergency' => 'اگر خونریزی‌ات زیاد شد، تب داری، درد شدید داری یا احساس می‌کنی ممکن است به خودت یا کودکت آسیب بزنی، همین حالا با پزشک یا اورژانس تماس بگیر.',

    // Static until L3-09 seeds the FAQ group `stage-postpartum` and the template reads it from the Faq context.
    'faq' => [
        'title' => 'سؤال‌های رایج پس از زایمان',
        'items' => [
            [
                'question' => 'ریتمی بیماری کودک را تشخیص می‌دهد؟',
                'answer' => 'نه. بخش کودک دفتر ثبت و یادآور است؛ برای هر نگرانی با پزشک کودک صحبت کن.',
            ],
            [
                'question' => 'برای دو فرزند هم کار می‌کند؟',
                'answer' => 'بله؛ برای هر فرزند پروفایل جدا بساز.',
            ],
            [
                'question' => 'همسرم می‌تواند ثبت کند؟',
                'answer' => 'بله، با دعوت او به‌عنوان همدم.',
            ],
        ],
    ],

    'app_cta' => [
        'title' => 'ریتمی را رایگان نصب کن',
        'lead' => 'در اپ، مرحله‌ات را انتخاب کن؛ بقیه‌اش با ماست.',
    ],

    'mock' => [
        'postpartum_baby' => [
            'label' => 'آوا · ۴ ماهه',
            'chart' => 'نمودار رشد · وزن',
            'rows' => [
                ['label' => 'واکسن ۴ ماهگی', 'value' => '۳ روز دیگر', 'tint' => 'luteal'],
                ['label' => 'شیر امروز', 'value' => '۶ بار', 'tint' => 'fertile'],
                ['label' => 'خواب', 'value' => '۱۴ ساعت', 'tint' => 'lilac'],
            ],
        ],
    ],
];
