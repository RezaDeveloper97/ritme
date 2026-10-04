<?php

declare(strict_types=1);

/*
 * /menopause — design/html/menopause.html. Structure (icons, colours, links, mock screens) lives in
 * App\Domain\Content\Stages\Menopause; the keys of `features`, `tools`, `help` and `hero.float` match its specs.
 * `mock` is the copy of the decorative phone screen mock/screens/menopause-status (demo values, aria-hidden).
 * Red lines: no diagnosis claims, no «حتماً/قطعاً/دقیق‌ترین/تضمینی», no sales pressure; treatment decisions stay
 * with the user and her doctor.
 */

return [
    'name' => 'یائسگی',

    'seo' => [
        'title' => 'یائسگی: ثبت گرگرفتگی، روند علائم و یادآور چکاپ',
        'description' => 'گرگرفتگی، خواب، حال و خونریزی‌های غیرمنتظره را ثبت کن، روند ماهانه علائمت را ببین و با گزارشی روشن و یادآور چکاپ‌ها پیش پزشک برو.',
    ],

    'hero' => [
        'eyebrow' => 'یائسگی',
        // h1 = title with the lilac `highlight` phrase in place of :highlight (design wording).
        'highlight' => 'فصل تازه',
        'title' => 'یائسگی، :highlight؛ نه پایان راه',
        'lead' => 'گرگرفتگی، خواب، حال و خونریزی‌های غیرمنتظره را ثبت کن؛ روند علائمت را ببین و با گزارشی روشن پیش پزشک برو.',
        'float' => [
            'hot-flash' => ['title' => 'گرگرفتگی امروز', 'text' => '۴ بار · ۲ کمتر از دیروز'],
            'mammogram' => ['title' => 'ماموگرافی', 'text' => 'یادآور آبان ۱۴۰۵'],
        ],
    ],

    'features' => [
        'symptoms' => [
            'eyebrow' => 'علائم',
            'title' => 'روند خودت را ببین، نه فقط امروز',
            'text' => 'امتیاز ماهانه علائم و روند گرگرفتگی و خواب؛ بر اساس ثبت‌های خودت، نه بازگرداندن همه‌چیز به چرخه.',
            'points' => ['ثبت سریع گرگرفتگی', 'امتیاز ماهانه علائم', 'روند ۶ ماهه'],
        ],
        'checkups' => [
            'eyebrow' => 'چکاپ‌ها',
            'title' => 'مراقبت‌هایی که این سال‌ها مهم‌تر می‌شوند',
            'text' => 'یادآور ماموگرافی، سنجش تراکم استخوان و فشار خون، با زمانی که با پزشکت هماهنگ کرده‌ای.',
            'points' => ['یادآور چکاپ‌ها', 'پرونده سلامت', 'گزارش ۳ ماهه برای پزشک'],
        ],
        'treatment' => [
            'eyebrow' => 'درمان',
            'title' => 'اگر درمان داری، پیگیری‌اش ساده باشد',
            'screen_title' => 'اگر درمان داری',
            'text' => 'یادآور دارو و ثبت اثر آن بر علائم؛ تصمیم درمان همیشه با تو و پزشکت است.',
            'points' => ['یادآور دارو', 'ثبت عوارض و تغییرات', 'بازبینی با پزشک'],
        ],
    ],

    'tools' => [
        'hot-flash' => ['title' => 'ثبت گرگرفتگی', 'text' => 'با یک لمس'],
        'report' => ['title' => 'گزارش برای پزشک', 'text' => 'خلاصه ۳ ماه'],
        'checkups' => ['title' => 'چک‌لیست چکاپ‌ها', 'text' => 'بر اساس سن و سابقه'],
    ],

    'help' => [
        'clinicians' => ['title' => 'پزشک و ماما', 'text' => 'ویزیت آنلاین یا حضوری با پرونده‌ای که خودت اجازه‌اش را می‌دهی.'],
        'pelvic-floor' => ['title' => 'کف لگن', 'text' => 'برنامه ۸ هفته‌ای تمرین‌های کف لگن.'],
        // The design text («پوشش زایمان») belongs to the pregnancy page; menopause gets a neutral line.
        'insurance' => ['title' => 'بیمه', 'text' => 'پوشش درمان و ثبت خسارت با مدارک پرونده.'],
    ],

    'emergency' => 'هر خونریزی یا لکه‌بینی بعد از یائسگی را ثبت کن و به پزشک خبر بده؛ اگر خونریزی زیاد است یا درد قفسه سینه داری، با اورژانس تماس بگیر.',

    // Static until L3-09 seeds the FAQ group `stage-menopause` and the template reads it from the Faq context.
    'faq' => [
        'title' => 'سؤال‌های رایج درباره یائسگی',
        'items' => [
            [
                'question' => 'از کجا بفهمم وارد یائسگی شده‌ام؟',
                'answer' => 'معمولاً وقتی ۱۲ ماه پیاپی پریود نشده باشی؛ برای اطمینان با پزشک صحبت کن.',
            ],
            [
                'question' => 'ریتمی درمان پیشنهاد می‌دهد؟',
                'answer' => 'نه. ریتمی کمک می‌کند علائمت را دقیق‌تر ببینی و با گزارش روشن پیش پزشک بروی.',
            ],
            [
                'question' => 'داده چرخه‌های قبلی‌ام می‌ماند؟',
                'answer' => 'بله؛ همه تاریخچه‌ات حفظ می‌شود.',
            ],
        ],
    ],

    'app_cta' => [
        'title' => 'ریتمی را رایگان نصب کن',
        'lead' => 'در اپ، مرحله‌ات را انتخاب کن؛ بقیه‌اش با ماست.',
    ],

    'mock' => [
        'menopause_status' => [
            'label' => 'حالت یائسگی',
            'title' => '۱۴ ماه بدون پریود',
            'stats' => [
                ['value' => '۴', 'label' => 'گرگرفتگی', 'tint' => 'period'],
                ['value' => '۱', 'label' => 'تعریق شبانه', 'tint' => 'lilac'],
                ['value' => '۵س', 'label' => 'خواب', 'tint' => 'fertile'],
            ],
            'score_label' => 'امتیاز علائم',
            'score' => '۱۴ از ۴۴',
            'action' => 'گرگرفتگی الان',
        ],
    ],
];
