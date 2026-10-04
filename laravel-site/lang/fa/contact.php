<?php

declare(strict_types=1);

/*
 * Copy of /contact (pages/contact.blade.php, ContactController, ContactRequest), from design/html/contact.html.
 * Channel values (emails, phone, hours, address) come from the contact settings (admin → تنظیمات → تماس); the FAQ
 * block is the `contact` FAQ group. Red lines: no diagnosis claims, no sales pressure.
 */

return [
    'seo' => [
        'title' => 'تماس با ریتمی؛ پشتیبانی، همکاری، رسانه و حریم خصوصی',
        'description' => 'راه‌های تماس با ریتمی: پشتیبانی داخل اپ، ایمیل، تلفن و فرم تماس برای پشتیبانی کاربر، همکاری کسب‌وکار، متخصصان سلامت، رسانه و حریم خصوصی.',
    ],

    'intro' => [
        'eyebrow' => 'تماس با ما',
        'title' => 'حرفت را بشنویم',
        'lead' => 'برای مشکل در اپ، سریع‌ترین راه گفت‌وگو با پشتیبانی داخل اپ است.',
    ],

    'channels' => [
        'label' => 'راه‌های تماس',
        'app' => [
            'title' => 'پشتیبانی داخل اپ',
            'path' => 'من › گفت‌وگو با پشتیبانی',
            'response' => 'پاسخ: :time',
        ],
        'email' => [
            'title' => 'ایمیل',
            'partnership' => 'برای همکاری: ',
        ],
        'phone' => ['title' => 'تلفن'],
        'address' => ['title' => 'نشانی'],
    ],

    'emergency' => 'اگر وضعیت اضطراری است، منتظر پاسخ ما نمان و با :number تماس بگیر.',

    'form' => [
        'title' => 'فرم تماس',
        'topic' => 'موضوع',
        'name' => 'نام',
        'name_placeholder' => 'نام تو',
        'contact' => 'ایمیل یا شماره همراه',
        'contact_placeholder' => 'برای پاسخ',
        'message' => 'پیام',
        'message_placeholder' => 'بنویس…',
        'note' => 'اطلاعات سلامت را در این فرم ننویس؛ برای مسائل حساب کاربری از پشتیبانی داخل اپ استفاده کن.',
        'submit' => 'ارسال پیام',
        'honeypot' => 'این فیلد را خالی بگذار',
        'sent_title' => 'پیامت رسید',
        'sent' => 'ممنون که نوشتی. پیامت به دست تیم ریتمی رسید و از همان راهی که گذاشتی جواب می‌دهیم.',
        'errors_title' => 'چند مورد را درست کن و دوباره بفرست:',
    ],

    'faq' => [
        'eyebrow' => 'سؤالات متداول',
        'title' => 'شاید جوابت اینجا باشد',
        'all' => 'همه سؤال‌ها',
    ],

    'validation' => [
        'topic_required' => 'موضوع پیام را انتخاب کن.',
        'topic_enum' => 'یکی از موضوع‌های فهرست را انتخاب کن.',
        'name_required' => 'نامت را بنویس.',
        'name_max' => 'نام حداکثر :name_max نویسه می‌تواند باشد.',
        'contact_required' => 'یک ایمیل یا شماره همراه بنویس تا بتوانیم جواب بدهیم.',
        'contact_max' => 'ایمیل یا شماره همراه بیش از حد طولانی است.',
        'contact_format' => 'یک ایمیل درست (مثل name@example.com) یا شماره همراه (مثل ۰۹۱۲۳۴۵۶۷۸۹) بنویس.',
        'message_required' => 'متن پیام را بنویس.',
        'message_min' => 'پیام کمی کوتاه است؛ دست‌کم :min نویسه بنویس.',
        'message_max' => 'پیام حداکثر :max نویسه می‌تواند باشد.',
        'expired' => 'از باز شدن این فرم زمان زیادی گذشته است؛ لطفاً دوباره «ارسال پیام» را بزن.',
    ],
];
