<?php

declare(strict_types=1);

/*
 * Copy of /about (pages/about.blade.php), from design/html/about.html. Icons and colours live in the view; words and
 * content data live here. Values in «[…]» are the design's placeholders (docs/AUDIT.md §5.2): the owner must replace
 * them with real facts — never invent numbers, names or dates.
 *
 * Team: one entry per person — `name`, `role`, `media` (media library id for the photo, or null → «[عکس]» circle).
 * Links: `url` null renders the card without a link (no href="#").
 */

return [
    'seo' => [
        'title' => 'درباره ریتمی؛ صدای زن، نه دفترچه راهنمای بدن زن',
        'description' => 'ریتمی را ساختیم تا کنار زنان باشیم؛ صادق درباره عدم قطعیت، بی‌قضاوت و با داده‌ای که مال خود توست. داستان، ارزش‌ها، خطوط قرمز و تیم ریتمی.',
    ],

    'hero' => [
        'eyebrow' => 'درباره ریتمی',
        'title_highlight' => 'صدای زن',
        'title_rest' => '، نه دفترچه راهنمای بدن زن',
        'lead' => 'ریتمی را ساختیم چون اپ‌های سلامت زنان یا پر از قطعیت‌های ساختگی بودند یا پر از ترس. ما می‌خواهیم کنارت باشیم؛ صادق، آرام و بی‌قضاوت.',
    ],

    'story' => [
        'eyebrow' => 'داستان ما',
        'title' => 'از یک سؤال ساده شروع شد',
        'paragraphs' => [
            '[داستان شکل‌گیری ریتمی به روایت بنیان‌گذار: چه مشکلی دیدید، چرا شروع کردید و ریتمی امروز کجاست.]',
            '[مأموریت در یک جمله]',
        ],
        'stats_label' => 'ریتمی در چند عدد',
        'stats' => [
            ['value' => '[عدد]', 'label' => 'کاربر فعال'],
            ['value' => '[عدد]', 'label' => 'ثبت روزانه در ماه'],
            ['value' => '[عدد]', 'label' => 'متخصص همکار'],
            ['value' => '[سال]', 'label' => 'سال تأسیس'],
        ],
    ],

    'values' => [
        'eyebrow' => 'ارزش‌ها',
        'title' => 'چطور تصمیم می‌گیریم',
        'items' => [
            ['title' => 'صادق درباره عدم قطعیت', 'text' => 'هر جا مطمئن نیستیم، همین را می‌گوییم.'],
            ['title' => 'بی‌قضاوت', 'text' => 'هیچ بدنی «مشکل‌دار» نیست؛ هر بدن ریتم خودش را دارد.'],
            ['title' => 'داده مال توست', 'text' => 'دیدن، اصلاح و خروجی گرفتن داده‌ات همیشه رایگان است.'],
        ],
    ],

    'red_lines' => [
        'eyebrow' => 'خطوط قرمز',
        'title' => 'کارهایی که هرگز نمی‌کنیم',
        'lead' => 'این‌ها قانون‌های داخلی ما هستند؛ هیچ فشار مالی یا زمانی عوضشان نمی‌کند.',
        'items' => [
            'داده سلامت تو را نمی‌فروشیم',
            'برای فروش بیشتر نمی‌ترسانیم',
            'تشخیص پزشکی نمی‌دهیم',
            'هشدارهای مهم سلامت را پولی نمی‌کنیم',
            'ابزارهای پایه را قفل نمی‌کنیم',
            'بدون اجازه جداگانه، از داده‌ات برای پیشنهاد کالا استفاده نمی‌کنیم',
        ],
    ],

    'team' => [
        'eyebrow' => 'تیم',
        'title' => 'آدم‌های پشت ریتمی',
        'photo_placeholder' => '[عکس]',
        'members' => [
            ['name' => '[نام]', 'role' => '[نقش]', 'media' => null],
            ['name' => '[نام]', 'role' => '[نقش]', 'media' => null],
            ['name' => '[نام]', 'role' => '[نقش]', 'media' => null],
            ['name' => '[نام]', 'role' => '[نقش]', 'media' => null],
        ],
        // E-E-A-T: the medical/editorial review policy (anchor #review-policy; articles can link to it).
        'council' => [
            'title' => 'شورای علمی',
            'text' => 'محتوای سلامت و قواعد هشدارها پیش از انتشار توسط [نام متخصصان: زنان و زایمان، مامایی، کودک] بازبینی می‌شود.',
        ],
    ],

    'links' => [
        'label' => 'بیشتر درباره ریتمی',
        'social' => ['title' => 'مسئولیت اجتماعی', 'text' => 'آگاهی، حق همه زنان است'],
        'careers' => ['title' => 'فرصت‌های شغلی', 'text' => '[موقعیت‌های باز]', 'url' => null],
        'press' => ['title' => 'رسانه و همکاری', 'text' => 'برای خبرنگاران و سازمان‌ها'],
    ],
];
