<?php

declare(strict_types=1);

/*
|--------------------------------------------------------------------------
| Persian validation messages
|--------------------------------------------------------------------------
|
| The API answers in the caller's language (`Accept-Language`, resolved by
| ResolvesLocale), but without this file every 422 fell back to Laravel's
| built-in English lines — so a Persian user marking a period saw
| "The start date field must be a date before or equal to today".
|
| Any key missing here still falls back to the English translation, so this
| file only needs the rules the API actually uses.
|
*/

return [
    'accepted' => 'فیلد :attribute باید پذیرفته شود.',
    'after' => ':attribute باید تاریخی بعد از :date باشد.',
    'after_or_equal' => ':attribute باید تاریخی برابر یا بعد از :date باشد.',
    'array' => ':attribute باید یک آرایه باشد.',
    'before' => ':attribute باید تاریخی قبل از :date باشد.',
    'before_or_equal' => ':attribute باید تاریخی برابر یا قبل از :date باشد.',
    'boolean' => 'فیلد :attribute باید درست یا نادرست باشد.',
    'confirmed' => 'تکرار :attribute با آن مطابقت ندارد.',
    'date' => ':attribute یک تاریخ معتبر نیست.',
    'date_format' => ':attribute با قالب :format مطابقت ندارد.',
    'different' => ':attribute و :other باید متفاوت باشند.',
    'digits' => ':attribute باید :digits رقم باشد.',
    'digits_between' => ':attribute باید بین :min و :max رقم باشد.',
    'email' => ':attribute باید یک ایمیل معتبر باشد.',
    'exists' => ':attribute انتخاب‌شده معتبر نیست.',
    'file' => ':attribute باید یک فایل باشد.',
    'filled' => 'فیلد :attribute نمی‌تواند خالی باشد.',
    'image' => ':attribute باید یک تصویر باشد.',
    'in' => ':attribute انتخاب‌شده معتبر نیست.',
    'integer' => ':attribute باید یک عدد صحیح باشد.',
    'max' => [
        'array' => ':attribute نباید بیشتر از :max مورد باشد.',
        'file' => 'حجم :attribute نباید بیشتر از :max کیلوبایت باشد.',
        'numeric' => ':attribute نباید بزرگ‌تر از :max باشد.',
        'string' => ':attribute نباید بیشتر از :max نویسه باشد.',
    ],
    'mimes' => ':attribute باید فایلی از نوع :values باشد.',
    'min' => [
        'array' => ':attribute باید حداقل :min مورد باشد.',
        'file' => 'حجم :attribute باید حداقل :min کیلوبایت باشد.',
        'numeric' => ':attribute باید حداقل :min باشد.',
        'string' => ':attribute باید حداقل :min نویسه باشد.',
    ],
    'not_in' => ':attribute انتخاب‌شده معتبر نیست.',
    'numeric' => ':attribute باید یک عدد باشد.',
    'prohibited' => 'فیلد :attribute مجاز نیست.',
    'regex' => 'قالب :attribute معتبر نیست.',
    'required' => 'فیلد :attribute الزامی است.',
    'required_if' => 'وقتی :other برابر :value است، فیلد :attribute الزامی است.',
    'required_with' => 'وقتی :values موجود است، فیلد :attribute الزامی است.',
    'required_without' => 'وقتی :values موجود نیست، فیلد :attribute الزامی است.',
    'same' => ':attribute و :other باید یکسان باشند.',
    'size' => [
        'array' => ':attribute باید :size مورد باشد.',
        'file' => 'حجم :attribute باید :size کیلوبایت باشد.',
        'numeric' => ':attribute باید برابر :size باشد.',
        'string' => ':attribute باید :size نویسه باشد.',
    ],
    'string' => ':attribute باید یک رشته باشد.',
    'unique' => ':attribute قبلاً ثبت شده است.',
    'url' => ':attribute باید یک نشانی معتبر باشد.',

    /*
    | Per-field overrides. The generic `before_or_equal` line would render the
    | rule's raw parameter ("today"), so the date fields the app sends get a
    | sentence that reads naturally in Persian instead.
    */
    'custom' => [
        'start_date' => [
            'before_or_equal' => 'تاریخ شروع نمی‌تواند در آینده باشد.',
        ],
        'end_date' => [
            'before_or_equal' => 'تاریخ پایان نمی‌تواند در آینده باشد.',
            'after_or_equal' => 'تاریخ پایان نمی‌تواند قبل از تاریخ شروع باشد.',
        ],
        'date' => [
            'before_or_equal' => 'این تاریخ نمی‌تواند در آینده باشد.',
        ],
        'log_date' => [
            'before_or_equal' => 'تاریخ ثبت نمی‌تواند در آینده باشد.',
        ],
        'last_period_start' => [
            'before_or_equal' => 'تاریخ شروع آخرین پریود نمی‌تواند در آینده باشد.',
        ],
        'lmp_date' => [
            'before_or_equal' => 'تاریخ اولین روز آخرین پریود نمی‌تواند در آینده باشد.',
        ],
        'ultrasound_date' => [
            'before_or_equal' => 'تاریخ سونوگرافی نمی‌تواند در آینده باشد.',
        ],
        'first_fetal_movement_date' => [
            'before_or_equal' => 'تاریخ اولین حرکت جنین نمی‌تواند در آینده باشد.',
        ],
    ],

    'attributes' => [
        'start_date' => 'تاریخ شروع',
        'end_date' => 'تاریخ پایان',
        'date' => 'تاریخ',
        'log_date' => 'تاریخ ثبت',
        'mobile' => 'شماره موبایل',
        'code' => 'کد تایید',
        'name' => 'نام',
        'birthday' => 'تاریخ تولد',
        'weight' => 'وزن',
        'height' => 'قد',
        'period_duration' => 'طول پریود',
        'cycle_duration' => 'طول سیکل',
        'last_period_start' => 'شروع آخرین پریود',
        'lmp_date' => 'اولین روز آخرین پریود',
        'ultrasound_date' => 'تاریخ سونوگرافی',
        'first_fetal_movement_date' => 'تاریخ اولین حرکت جنین',
        'week' => 'هفته',
        'note' => 'یادداشت',
        'notes' => 'یادداشت‌ها',
    ],
];
