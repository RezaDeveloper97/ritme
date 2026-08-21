# CLAUDE.MD

## قوانین کدنویسی (الزامی)

**SOLID + DRY + KISS + YAGNI** در همه جا رعایت شود.

## API

- ورژن‌بندی: `/api/v1/*`
- هر endpoint جدید **باید** در Swagger مستند شود (`@OA\`)

## معماری: DDD

```
app/
├── Domain/           # Entities, ValueObjects, Repository Interfaces
├── Application/      # DTOs, UseCases, Services
└── Infrastructure/   # Repository Implementations
```

## چندزبانگی (الزامی — هیچ استثنایی ندارد)

**فهرست زبان‌های اپ داده است، نه کد.** زبان‌ها در جدول `languages` نگهداری
می‌شوند و ادمین از `/admin/languages` اضافه/غیرفعال/پیش‌فرض‌شان می‌کند. فارسی
(`fa`) و انگلیسی (`en`) فقط دو ردیفِ seed شده‌اند — هیچ جای کد نباید زبان‌ها را
دستی بشمارد.

```php
// ❌ هرگز
$locale = $request->header('Accept-Language', 'fa');
'title.fa' => ['required', 'string'],
$item->title['fa'] ?? '—'
in_array($code, ['fa', 'en'], true)

// ✅ همیشه
$locale = $this->resolveLocale($request);            // ResolvesLocale trait
...Translatable::rules('title', required: true),     // یک ورودی برای هر زبان فعال
$item->localized('title')                            // با fallback
app(LanguageRegistry::class)->codes()                // فهرست زنده
```

### قطعات

| کار | جای درست |
| --- | --- |
| فهرست/پیش‌فرض/جهت زبان‌ها | `App\Services\Language\LanguageRegistry` (cache دارد؛ بعد از هر نوشتن `flush()`) |
| قواعد اعتبارسنجی فیلد چندزبانه | `App\Support\Translatable::rules()` / `rulesFor()` |
| خواندن مقدار چندزبانه | `HasLocalizedContent::localized()` یا `Translatable::pick()` |
| فایل‌های رشته‌های رابط کاربری | `App\Services\Language\TranslationStore` |
| ساخت زبان جدید (فایل‌ها + پیام‌های هوشمند) | `App\Services\Language\LanguageProvisioner` |
| تعیین locale هر درخواست | middleware `SetLocale` (روی گروه `api`؛ پنل ادمین `setlocale:default`) |

### قواعد

- ستون‌های چندزبانه JSON‌اند با کلید کدِ زبان: `{"fa": "...", "en": "...", "ar": "..."}`.
  تعداد کلیدها با جدول `languages` رشد می‌کند.
- **فقط زبان پیش‌فرض `required` است.** بقیه اختیاری‌اند، چون هر خواننده‌ای به
  زبان پیش‌فرض fallback می‌کند. محتوا باید قبل از ترجمه‌شدن قابل انتشار باشد.
- فرم‌های ادمین از `<x-admin.translatable>` استفاده می‌کنند؛ این کامپوننت برای هر
  زبانِ فعال یک ورودی (با `dir` درست) می‌سازد. هرگز فیلد فارسی/انگلیسی دستی نسازید.
- ساخت یک زبان، فایل‌های `storage/app/translations/<code>/` (کپی از زبان پیش‌فرض)،
  `lang/<code>/` و ردیف‌های `message_contents` آن زبان را می‌سازد. این پوشه روی
  volume ماندگار `backend-storage` است، پس با هر deploy پاک نمی‌شود.
- `resources/translations/` seedِ داخل ریپو است. بعد از تغییر کلیدها در
  `frontend/messages/**` دستور `php artisan translations:import` را بزنید تا
  زبانی که فردا ساخته می‌شود کلیدهای جدید را داشته باشد.
- برچسب‌های enum (`label('fa')`) برای زبان‌های تازه به انگلیسی برمی‌گردند؛ اگر
  متن آن زبان مهم است، آن رشته را به `message_contents` یا فایل ترجمه ببرید.
- `GET /languages` و `GET /languages/{code}/messages` عمومی‌اند — فرانت‌اند در
  زمان اجرا از همین‌ها زبان‌ها را می‌خواند، برای همین اضافه‌کردن زبان به build
  دوباره‌ی فرانت نیاز ندارد.

جزئیات سمت فرانت‌اند: `frontend/CLAUDE.md` §6.
