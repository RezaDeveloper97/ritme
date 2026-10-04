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

    // Product page (L6-03). No «فروشنده بررسی‌شده» / sales counts / verified-buyer claims until they are real.
    'product' => [
        'seo_title' => ':name؛ قیمت و خرید',
        'seo_description' => ':name از :brand در فروشگاه ریتمی: قیمت، سایز، موجودی و مشخصات کامل، با پرداخت در محل.',
        'seo_description_plain' => ':name در فروشگاه ریتمی: قیمت، سایز، موجودی و مشخصات کامل، با پرداخت در محل.',
        'sales' => ':count+ خرید',
        'demo_note' => 'این کالا و قیمتش نمونه نمایشی فروشگاه است.',
        'gallery' => [
            'label' => 'تصاویر :name',
            'image' => ':name — تصویر :n',
            'thumbs' => 'تصاویر دیگر',
            'close' => 'بستن',
            'previous' => 'تصویر قبلی',
            'next' => 'تصویر بعدی',
            'counter' => 'تصویر :n از :total',
        ],
        'color' => 'رنگ',
        'color_label' => 'رنگ: :name',
        'size' => 'سایز',
        'size_chart_link' => 'جدول سایز',
        'size_soldout' => ':size (ناموجود)',
        'quantity' => 'تعداد',
        'more' => 'بیشتر',
        'less' => 'کمتر',
        'add' => 'افزودن به سبد',
        'unavailable' => 'ناموجود',
        'cart_soon' => 'افزودن به سبد به‌زودی فعال می‌شود.',
        'wishlist' => 'افزودن :name به علاقه‌مندی‌ها',
        'delivery' => [
            ['icon' => 'truck', 'title' => 'ارسال', 'text' => 'تحویل ۲ تا ۴ روز کاری · پرداخت در محل'],
            ['icon' => 'return', 'title' => 'بازگشت', 'text' => 'تا ۷ روز برای کالای باز نشده'],
            ['icon' => 'package', 'title' => 'بسته‌بندی ساده', 'text' => 'بدون نام محصول روی جعبه'],
        ],
        'specs' => 'مشخصات',
        'description' => 'توضیحات',
        'size_chart' => 'جدول سایز',
        'size_chart_caption' => 'جدول سایز :name',
        'reviews' => [
            'title' => 'نظر خریداران',
            'lead' => 'نظرها پس از بررسی منتشر می‌شوند.',
            'summary' => 'میانگین :value از ۵ · :count نظر',
            'empty' => 'هنوز نظری برای این کالا ثبت نشده است.',
            'verified' => 'خرید تأییدشده',
            'demo' => 'نمونه',
            'demo_note' => 'نظرهای «نمونه» نمایشی‌اند و در امتیاز کالا حساب نمی‌شوند.',
            'pagination' => 'صفحه‌های نظرها',
            'previous' => 'نظرهای جدیدتر',
            'next' => 'نظرهای قدیمی‌تر',
            'page' => 'نظرها، صفحه :page',
            'page_of' => 'صفحه :page از :last',
        ],
        'review' => [
            'open' => 'نوشتن نظر درباره این کالا',
            'intro' => 'نظرت پس از بررسی منتشر می‌شود. شماره تماس یا اطلاعات شخصی ننویس.',
            'name' => 'نام نمایشی',
            'name_hint' => 'مثلاً «مادر آوا»',
            'rating' => 'امتیاز',
            'stars' => ':n ستاره',
            'size' => 'سایزی که خریدی',
            'size_none' => 'انتخاب نشده',
            'size_label' => 'سایز :size',
            'body' => 'متن نظر',
            'submit' => 'ارسال نظر',
            'sent' => 'ممنون! نظرت ثبت شد و پس از بررسی منتشر می‌شود.',
            'honeypot' => 'این فیلد را خالی بگذار',
            'errors' => [
                'title' => 'لطفاً موارد مشخص‌شده را اصلاح کن.',
                'name' => 'نام نمایشی را بین ۲ تا ۸۰ حرف بنویس.',
                'rating' => 'یک امتیاز از ۱ تا ۵ انتخاب کن.',
                'body' => 'متن نظر را بنویس (حداکثر ۲۰۰۰ حرف).',
                'body_short' => 'متن نظر دست‌کم ۱۰ حرف باشد.',
                'size' => 'سایز انتخاب‌شده در فهرست این کالا نیست.',
            ],
        ],
        'related' => 'معمولاً با این می‌خرند',
    ],
];
