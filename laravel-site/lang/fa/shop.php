<?php

declare(strict_types=1);

/*
 * Shop UI copy (L6-02: shop home + category listing). Category names / intros come from the catalog (admin); the
 * per-department promo copy below is keyed by the root category slug and falls back to `department` for roots the
 * admin adds later. Red lines: no superlatives, no pressure, no invented ratings / sales counts; single seller, so
 * cards show the brand.
 */
return [
    'name' => 'فروشگاه',
    'eyebrow' => 'فروشگاه ریتمی',
    'page_suffix' => 'صفحه :page',

    'home' => [
        'title' => 'آنچه برای خودت و کودکت لازم است',
        'lead' => 'سیسمونی، لباس نوزاد و محصولات آرایشی و بهداشتی از برندهای بررسی‌شده. فروشگاه از بخش سلامت جداست و داده سلامت تو برای پیشنهاد کالا استفاده نمی‌شود.',
        'seo_title' => 'فروشگاه ریتمی — سیسمونی، لباس نوزاد و محصولات بهداشتی',
        'seo_description' => 'سیسمونی، لباس نوزاد، تغذیه و خواب کودک و محصولات آرایشی و بهداشتی بانوان با جدول سایز، ترکیبات کامل و پرداخت در محل.',
        'categories_title' => 'دسته‌های :name',
        'products_title' => 'منتخب :name',
        'products_label' => 'محصولات :name',
    ],

    // Promo tiles of the two departments (shop.html), keyed by root category slug.
    'departments' => [
        'baby' => [
            'title' => 'برای آمدن نوزاد آماده شو',
            'products_title' => 'منتخب لباس و سیسمونی نوزاد',
            'products_lead' => null,
        ],
        'beauty' => [
            'title' => 'مراقبت از خودت، هر روز',
            'products_title' => 'مراقبت پوست و بهداشت بانوان',
            'products_lead' => 'هر محصول با شناسه IRC و فهرست کامل ترکیبات',
        ],
    ],
    'department' => [
        'cta' => 'ورود به فروشگاه',
        'all' => 'همه',
        'all_label' => 'همه محصولات :name',
    ],

    // The design's app-only features, as app CTAs (no fake progress or switches on the website).
    'app' => [
        'title' => 'در اپ ریتمی',
        'checklist' => [
            'title' => 'لیست سیسمونی',
            'meta' => 'فهرست پیشنهادی · هر چیزی را حذف یا اضافه کن',
            'text' => 'لیست را در اپ بساز و با خانواده به اشتراک بگذار تا هدیه‌ها تکراری نشوند.',
            'cta' => 'ساختن لیست در اپ',
        ],
        'reminder' => [
            'title' => 'یادآور خرید قبل از پریود',
            'meta' => 'فقط در اپ و فقط با اجازه خودت',
            'status' => 'پیش‌فرض خاموش',
            'text' => 'پیش‌فرض خاموش است. اگر در اپ روشنش کنی، فقط تاریخ تقریبی پریود بعدی برای زمان یادآور استفاده می‌شود و به هیچ فروشنده‌ای نمی‌رسد.',
        ],
    ],

    'trust' => [
        'title' => 'خرید از فروشگاه ریتمی',
        'items' => [
            ['icon' => 'shield-check', 'title' => 'اصالت کالا', 'text' => 'فقط برندهای بررسی‌شده'],
            ['icon' => 'return', 'title' => '۷ روز بازگشت', 'text' => 'برای کالای باز نشده'],
            ['icon' => 'truck', 'title' => 'پرداخت در محل', 'text' => 'هزینه را هنگام تحویل بپرداز'],
            ['icon' => 'package', 'title' => 'بسته‌بندی ساده', 'text' => 'بدون نام محصول روی جعبه'],
        ],
    ],

    'demo_note' => 'کالاها و قیمت‌های این صفحه نمونه نمایشی فروشگاه‌اند.',
    'empty' => 'هنوز کالایی در فروشگاه نیست. به‌زودی محصولات اضافه می‌شوند.',

    'cart' => [
        'label' => 'سبد خرید',
        'count' => ':count کالا در سبد',
    ],

    'subnav' => [
        'departments' => 'بخش‌های فروشگاه',
        'categories' => 'دسته‌های :name',
    ],

    'category' => [
        'seo_title' => ':name؛ قیمت و خرید آنلاین',
        'seo_description' => ':name در فروشگاه ریتمی: قیمت، برند، سایز و موجودی هر کالا را ببین و با پرداخت در محل سفارش بده.',
        'count' => ':count کالا',
        'count_filtered' => ':count کالا با فیلترهای فعلی',
        'results' => 'کالاهای :name',
        'empty' => 'کالایی با این فیلترها پیدا نشد.',
        'empty_category' => 'هنوز کالایی در این دسته نیست.',
        'reset' => 'پاک کردن فیلترها',
    ],

    'sort' => [
        'label' => 'ترتیب:',
        'menu' => 'ترتیب نمایش کالاها',
    ],

    'filters' => [
        'title' => 'فیلترها',
        'form' => 'فیلتر کالاها',
        'clear' => 'پاک کردن',
        'categories' => 'دسته',
        'sizes' => 'سایز',
        'colors' => 'رنگ',
        'color' => 'رنگ :name',
        'price' => 'قیمت (تومان)',
        'price_min' => 'از',
        'price_max' => 'تا',
        'price_range' => 'قیمت کالاهای این دسته از :min تا :max تومان',
        'brands' => 'برند',
        'stock' => 'فقط کالاهای موجود',
        'apply' => 'اعمال فیلترها',
        'active' => 'فیلترهای فعال',
        'remove' => 'حذف فیلتر :name',
        'price_from' => 'از :amount',
        'price_to' => 'تا :amount',
        'in_stock' => 'فقط موجود',
    ],

    'pagination' => [
        'label' => 'صفحه‌های کالاها',
        'previous' => 'قبلی',
        'next' => 'بعدی',
        'page' => 'صفحه :page',
    ],
];
