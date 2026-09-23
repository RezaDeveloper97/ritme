# CLAUDE.MD

> **⛔ FROZEN — production maintenance only (since 2026-09-23).** All new backend work goes to **`backend-go/`**
> (read `backend-go/CLAUDE.md`, skills `new-endpoint` / `local-dev`). This Laravel app stays only because
> production still runs it until the cutover (T-M2-26/27); staging runs Go only (T-M2-28).
> Allowed here: production bug fixes (mirror the fix in backend-go), and the **mirror Laravel migration** that
> every goose schema change needs until T-M2-27 (`docs/go-migration/migrations.md`). No new features, routes or
> content types.
>
> **⛔ منجمد — فقط نگهداری production.** هر کار جدید بک‌اند در `backend-go/` انجام می‌شود؛ اینجا فقط رفع باگ
> production و migrationِ آینه‌ی تغییرات schema تا T-M2-27.

## قوانین کدنویسی (الزامی)

**SOLID + DRY + KISS + YAGNI** در همه جا رعایت شود.

## API

- ورژن‌بندی: `/api/v1/*`
- هر endpoint جدید **باید** در Swagger مستند شود (`@OA\`)

## نشست و توکن (الزامی)

کاربر باید یک سال وارد بماند. پیش‌زمینه در `docs/investigations/session-logout.md` آمده است.

- **عمر توکن:** ۳۶۵ روز ثابت (`config/passport.php` → `token_lifetime_days`، که در `AppServiceProvider` به `P365D` تبدیل می‌شود). آن را به `P1Y` یا عدد کمتر برنگردانید. تست‌های `tests/Feature/Auth/SessionTokenTest.php` جلوی این را می‌گیرند.
- **تمدید:** `POST /api/v1/auth/refresh-session` فقط وقتی کمتر از `refresh_window_days` (۳۰ روز) به انقضای توکن مانده باشد، توکن تازه‌ی ۳۶۵ روزه می‌دهد و توکن قبلی را revoke می‌کند (اول صدور، بعد revoke). در غیر این صورت `refreshed: false` برمی‌گرداند و کاری نمی‌کند.
- **قرارداد 401:** 401ِ احراز هویت (`AuthenticationException` روی `/api`) همیشه JSON است و فیلد `error_code` دارد که یکی از `token_revoked`، `token_expired` یا `unauthenticated` است (`bootstrap/app.php`). کلاینت فقط با همین 401ها نشست را پاک می‌کند، پس برای خطاهای دیگر (دسترسی، اعتبارسنجی، محدودیت) **هرگز** 401 برنگردانید و به‌جایش از 403/422/429 استفاده کنید.
- **revoke** فقط در سه جا مجاز است: logout، بلاک‌کردن کاربر توسط ادمین و حذف حساب. همین‌طور `refresh-session` که توکن قبلی را revoke می‌کند. `passport:purge` زمان‌بندی‌شده اضافه نکنید.
- **فلگ `ritme_auth`** را route فرانت‌اند (`POST /api/session/flag`) تنظیم می‌کند، نه بک‌اند. توکن هرگز در کوکی نمی‌رود.
- **کلیدها:** `storage/oauth-*.key` هرگز ship نشود. این فایل‌ها در `.dockerignore` و rsyncِ `deploy.sh` و `deploy-stage.sh` exclude شده‌اند (`--delete-excluded` اضافه نکنید). کلیدها فقط روی volumeِ `backend-storage` هستند. عوض‌کردن کلید همه‌ی کاربران را از حساب خارج می‌کند، پس فقط با برنامه انجام شود.

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
