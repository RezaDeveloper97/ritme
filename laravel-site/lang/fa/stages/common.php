<?php

declare(strict_types=1);

/*
 * Copy shared by every life-stage page (pages/stages/show.blade.php). Per-stage copy: lang/fa/stages/<slug>.php.
 */

return [
    'nav_label' => 'مرحله‌های زندگی',
    'breadcrumb' => 'مرحله‌ها',

    'hero' => [
        'download' => 'دانلود رایگان ریتمی',
        'how' => 'چطور کار می‌کند؟',
    ],

    'tools' => [
        'eyebrow' => 'ابزارهای این مرحله',
        'title' => 'کارهای کوچک، آمادگی بیشتر',
        'on_site' => 'روی سایت',
        'in_app' => 'در اپ',
    ],

    'help' => [
        'eyebrow' => 'خدمات مرتبط',
        'title' => 'وقتی کمک بیشتری لازم داری',
        'cta' => 'بیشتر',
    ],

    'readings' => [
        'eyebrow' => 'مجله',
        'title' => 'برای همین مرحله',
        'more' => 'همه',
    ],

    'faq' => [
        'eyebrow' => 'سؤال‌های رایج',
    ],

    // Phone mock-up screens (decorative, aria-hidden): resources/views/pages/stages/mock/screens/*. Keys = screen
    // name with underscores; a stage file's own `mock.<screen>` wins over these.
    'mock' => [
        'cycle_today' => [
            'day' => 'امروز · روز ۱۴',
            'countdown' => '۱۴ روز',
            'countdown_label' => 'تا پریود بعدی',
            'fertile' => 'پنجره باروری',
            'fertile_value' => 'امروز',
            'checkin' => 'امروز چطور بودی؟',
        ],
        // mock/screens/companion — shared with the home page «همدم» split; a stage may override it.
        'companion' => [
            'eyebrow' => 'همدم سارا',
            'title' => 'سلام علی',
            'card_label' => 'سیکل سارا',
            'card_value' => 'روز ۲۲ · لوتئال',
            'tip' => 'امروز: یک شام سبک و گرم آماده کن',
            'row_label' => 'قرص آهن سارا',
            'row_value' => '۲۱:۰۰',
        ],
        'continue' => 'ادامه',
    ],
];
