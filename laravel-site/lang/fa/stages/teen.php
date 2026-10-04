<?php

declare(strict_types=1);

/*
 * /teen — design/html/teen.html. Structure (icons, colours, links, mock screens) lives in
 * App\Domain\Content\Stages\Teen; the keys of `features`, `tools` and `hero.float` match its specs. No `help`
 * block: the teen version has no shop or services.
 * `mock` overrides the copy of the shared decorative phone screens cycle-today and companion (aria-hidden),
 * departing from the design on purpose: no fertile window and no partner copy on a page for young teens.
 * Red lines: age-appropriate, no diagnosis claims, no «حتماً/قطعاً/دقیق‌ترین/تضمینی», no sales pressure; the mother
 * sees only what the teen allows.
 */

return [
    'name' => 'نوجوان و والدین',

    'seo' => [
        'title' => 'اولین پریود؛ تقویم ساده نوجوان با همراهی مادر',
        'description' => 'نسخه ساده و آموزشی ریتمی برای دختران نوجوان: تقویم پریود، کیف اضطراری مدرسه و جواب سؤال‌ها؛ بدون فروشگاه و تبلیغ، با همراهی مادر.',
    ],

    'hero' => [
        'eyebrow' => 'نوجوان و والدین',
        // h1 = title with the lilac `highlight` phrase in place of :highlight (design wording).
        'highlight' => 'بدون ترس',
        'title' => 'اولین پریود، :highlight و خجالت',
        'lead' => 'نسخه ساده و آموزشی ریتمی برای دخترهای نوجوان؛ بدون فروشگاه و تبلیغ، با امکان همراهی مادر.',
        'float' => [
            'next' => ['title' => 'پریود بعدی', 'text' => '۱۴ روز دیگر'],
            'voice' => ['title' => 'ثبت با صدا', 'text' => '۶ مورد ثبت شد'],
        ],
    ],

    'features' => [
        'simple' => [
            'eyebrow' => 'ساده',
            'title' => 'همان چیزی که یک نوجوان لازم دارد',
            'text' => 'تقویم ساده، کیف اضطراری مدرسه و جواب سؤال‌هایی که شاید روی پرسیدنش را نداشته باشد.',
            'points' => ['تقویم ساده پریود', 'کیف اضطراری مدرسه', '«طبیعی است؟» با زبان ساده'],
        ],
        'mother' => [
            'eyebrow' => 'همراهی مادر',
            'title' => 'مادر در جریان است، نه ناظر',
            'text' => 'مادر فقط چیزهایی را می‌بیند که دخترش اجازه بدهد؛ مثل نزدیک بودن پریود یا یادآور کیف.',
            'points' => ['اجازه با خود نوجوان', 'بدون نمایش جزئیات', 'راهنمای گفت‌وگو برای مادر'],
        ],
    ],

    'tools' => [
        'school-kit' => ['title' => 'کیف اضطراری مدرسه', 'text' => 'چه چیزهایی همراهت باشد'],
        'first-period' => ['title' => 'راهنمای اولین پریود', 'text' => 'برای دختر و مادر'],
        'calendar' => ['title' => 'تقویم ساده', 'text' => 'بدون پیچیدگی'],
    ],

    'emergency' => 'اگر خونریزی خیلی زیاد است، درد شدید داری یا تا ۱۵ سالگی پریود نشده‌ای، با مادرت یا پزشک صحبت کن.',

    // Static until L3-09 seeds the FAQ group `stage-teen` and the template reads it from the Faq context.
    'faq' => [
        'title' => 'سؤال‌های رایج نوجوان‌ها و والدین',
        'items' => [
            [
                'question' => 'نسخه نوجوان تبلیغ دارد؟',
                'answer' => 'نه؛ فروشگاه و هیچ پیشنهاد خریدی در آن نیست.',
            ],
            [
                'question' => 'مادرم همه چیز را می‌بیند؟',
                'answer' => 'نه؛ فقط چیزهایی که خودت اجازه بدهی.',
            ],
            [
                // Design placeholder «[سن و شرط رضایت والدین طبق قوانین]»: the real age and consent rule is
                // legal copy (open item, L3-09 FAQ admin) — no invented age here.
                'question' => 'از چند سالگی می‌شود استفاده کرد؟',
                'answer' => 'شرط سنی و رضایت والدین در قوانین استفاده و حریم خصوصی ریتمی آمده است؛ پیشنهاد ما این است که نصب و شروع کار با همراهی مادر یا والدین باشد.',
            ],
        ],
    ],

    'app_cta' => [
        'title' => 'ریتمی را رایگان نصب کن',
        'lead' => 'در اپ، مرحله‌ات را انتخاب کن؛ بقیه‌اش با ماست.',
    ],

    'mock' => [
        // Overrides stages/common.mock.cycle_today (same keys) — no fertile window for teens.
        'cycle_today' => [
            'day' => 'امروز · روز ۱۴',
            'countdown' => '۱۴ روز',
            'countdown_label' => 'تا پریود بعدی',
            'fertile' => 'کیف اضطراری مدرسه',
            'fertile_value' => 'آماده',
            'checkin' => 'امروز چطور بودی؟',
        ],
        // Overrides stages/common.mock.companion (same keys) — the mother's view, no partner copy for teens.
        'companion' => [
            'eyebrow' => 'همراهی مادر',
            'title' => 'سلام مادر',
            'card_label' => 'با اجازه دخترت',
            'card_value' => 'پریود نزدیک است',
            'tip' => 'امروز: یادآوری آرام کیف مدرسه',
            'row_label' => 'کیف اضطراری',
            'row_value' => 'آماده',
        ],
    ],
];
