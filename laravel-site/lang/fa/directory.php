<?php

declare(strict_types=1);

/*
 * Mother & child directory UI copy (L5-02 listing). Landing h1 / title / description / intro per city and
 * city × category come from Directory\Support\LandingCopy (admin-editable rows); the strings here are the /directory
 * page and the listing chrome. Red lines: no superlatives, no pressure, no paid placement.
 */
return [
    'name' => 'خدمات مادر و کودک',
    'eyebrow' => 'خدمات مادر و کودک',
    'all' => 'همه',
    'page_suffix' => 'صفحه :page',

    'index' => [
        'title' => 'جاهای خوب برای تو و کودکت، نزدیک خانه',
        'lead' => 'کلاس مادر و کودک، استخر، خانه بازی و کارگاه‌ها را ببین، صفحه هر مجموعه را بخوان و همین‌جا وقت رزرو کن.',
        'seo_title' => 'خدمات مادر و کودک — کلاس، استخر و خانه بازی نزدیک تو',
        'seo_description' => 'کلاس مادر و کودک، استخر، خانه بازی، موسیقی و کارگاه‌های فرزندپروری را با ساعت کاری، رده سنی و قیمت ببین و وقت رزرو کن.',
    ],
    'city' => [
        'lead' => 'مجموعه‌های مادر و کودک :city را ببین؛ ساعت کاری، رده سنی، خدمات و قیمت هر مجموعه در صفحه خودش است.',
    ],
    'category' => [
        'lead' => ':category در :city: ساعت کاری، رده سنی، خدمات و قیمت هر مجموعه را ببین و وقت رزرو کن.',
    ],

    'search' => [
        'label' => 'جست‌وجوی مجموعه‌ها',
        'what' => 'چه خدمتی؟',
        'what_placeholder' => 'استخر، کلاس، خانه بازی…',
        'where' => 'کجا؟',
        'where_any' => 'همه شهرها',
        'age' => 'سن کودک',
        'age_any' => 'هر سنی',
        'months' => ':n ماه',
        'years' => ':n سال',
        'submit' => 'جست‌وجو',
    ],
    'chips_label' => 'دسته‌های خدمات',

    'filters' => [
        'label' => 'فیلترها',
        'panel' => 'امکانات مجموعه',
        'apply' => 'اعمال فیلترها',
        'open' => 'باز است',
        'price' => 'قیمت',
        'quick' => 'فیلترهای سریع',
    ],
    'sort' => [
        'label' => 'ترتیب:',
        'menu' => 'ترتیب نتایج',
    ],
    'results' => [
        'heading' => 'نتیجه‌ها',
        'count' => ':count مجموعه',
        'where' => 'در :where',
        'empty' => 'با این فیلترها مجموعه‌ای پیدا نکردیم. فیلترها را کمتر کن یا دسته دیگری را ببین.',
        'reset' => 'پاک کردن فیلترها',
    ],
    'card' => [
        'ages' => 'مناسب :ages',
        'open' => 'باز است',
        'distance' => ':km کیلومتر',
    ],
    'pagination' => [
        'label' => 'صفحه‌های نتیجه',
        'page' => 'صفحه :page',
        'previous' => 'قبلی',
        'next' => 'بعدی',
    ],
    'fair' => 'ترتیب نتایج بر اساس سن کودک، فاصله و نظر مادرهاست. هیچ مجموعه‌ای برای بالاتر آمدن پول نمی‌دهد و آگهی ویژه نداریم.',

    'areas' => [
        'title' => 'مجموعه‌ها در شهر و محله تو',
        'cities' => 'شهرها',
        'districts' => 'محله‌های :city',
        'landings' => 'دسته‌ها',
        'landing' => ':category در :city',
        'note' => 'روی این صفحه نقشه نیست؛ آدرس هر مجموعه و دکمه «باز کردن در نقشه» در صفحه خودش است.',
    ],

    'business' => [
        'title' => 'مجموعه‌ای برای مادر و کودک داری؟',
        'text' => 'صفحه مجموعه‌ات را بساز، وقت‌های خالی را اعلام کن و رزرو بگیر.',
        'cta' => 'درخواست ثبت مجموعه',
    ],
];
