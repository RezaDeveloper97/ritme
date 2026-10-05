# راهنمای استقرار سایت ریتمی روی هاست cPanel (L10-01)

این راهنما سایت لاراولی ریتمی (`laravel-site/`) را روی هاست اشتراکی cPanel راه‌اندازی می‌کند. روی هاست **Node لازم نیست**
(بستهٔ آماده شامل خروجی build است)، **symlink لازم نیست** (رسانه‌ها مستقیم در `<public>/media` نوشته می‌شوند) و
**Supervisor لازم نیست** (یک خط cron هم زمان‌بند و هم صف را اجرا می‌کند).

> فهرست کنترل امنیتی پیش از انتشار: `docs/SECURITY.md` بخش ۵. عملکرد و critical CSS: `docs/PERFORMANCE.md`.

---

## ۰. پیش‌نیازهای هاست

| مورد | مقدار لازم |
|---|---|
| PHP | **8.2 یا بالاتر** (بسته با `config.platform.php = 8.2` ساخته می‌شود) |
| افزونه‌های PHP | `intl`، `mbstring`، `pdo_mysql`، `fileinfo`، `exif`، `gd` (یا `imagick`)، `openssl`، `curl`، `dom`، `xml`، `ctype`، `tokenizer` — پیشنهادی: `zip`، `opcache` |
| تابع‌ها | `proc_open` نباید در `disable_functions` باشد (زمان‌بند با آن فرمان‌ها را اجرا می‌کند) |
| پایگاه داده | MySQL 5.7+ یا MariaDB 10.3+ با `utf8mb4` |
| وب‌سرور | Apache یا LiteSpeed با `mod_rewrite` و اجازهٔ `.htaccess` (`AllowOverride`) |
| دسترسی | **Terminal** در cPanel (یا SSH) برای `php artisan` — بدون آن، بخش «نصب بدون Terminal» را ببینید |
| دیگر | Cron Jobs، AutoSSL، حداقل ۲۵۰ مگابایت فضا (بسته ≈ ۲۲ مگابایت فشرده، ≈ ۱۰۰ مگابایت باز) |

`php artisan app:install --check` همهٔ این موارد را روی خود هاست بررسی می‌کند.

---

## ۱. ساخت بسته (روی سیستم توسعه، نه روی هاست)

```bash
cd laravel-site
bash deploy/build-cpanel.sh --dry-run               # فقط بررسی ابزارها و فهرست فایل‌ها
bash deploy/build-cpanel.sh                         # چیدمان «ریشهٔ سند → public» یا «همه در public_html»
bash deploy/build-cpanel.sh --layout=public_html    # چیدمان «public_html کنار پوشهٔ برنامه»
bash deploy/build-cpanel.sh --layout=all            # هر دو بسته از یک build
```

- منبع پیش‌فرض یک خروجی تمیز از گیت است (`git archive HEAD`؛ با `--ref=<commit>` قابل تغییر). `--worktree` فقط برای
  آزمایش محلی است (فایل‌های commit نشده را هم برمی‌دارد و نام بسته `-dirty` می‌گیرد).
- مراحل: `composer install --no-dev --optimize-autoloader --classmap-authoritative` → پایگاه SQLite موقت برای رندر
  صفحه‌ها → `npm ci` → `CRITICAL_STRICT=1 npm run build` (Vite + `public/build/build-id.json` + service worker
  `public/sw.js` + critical CSS در `public/build/critical/`) → `php artisan filament:assets` → حذف فایل‌های توسعه
  (`tests`، `design`، `tasks`، `docs`، `tools`، `node_modules`، `.env`، SQLite، `resources/{css,js,fonts}`، پیکربندی‌های
  توسعه) → zip + فایل `sha256`.
- خروجی: `dist/ritme-site-<build-id>.zip` و `dist/ritme-site-<build-id>-public_html.zip` (پوشهٔ `dist/` در گیت نیست).
- ابزار لازم روی سیستم سازنده: git، PHP ≥ 8.2، composer، node + npm، zip، shasum و **Google Chrome** (برای critical CSS؛
  مسیر دیگر با `CHROME_PATH`). بدون Chrome ساخت متوقف می‌شود؛ `--allow-no-critical` فقط برای آزمایش است.
- اسکریپت هیچ چیزی آپلود نمی‌کند؛ بارگذاری روی هاست دستی و با تأیید مالک انجام می‌شود.

پیش از آپلود، checksum را مقایسه کنید: `shasum -a 256 -c ritme-site-<build-id>.zip.sha256`.

---

## ۲. انتخاب چیدمان

| چیدمان | کی؟ | ساختار روی هاست |
|---|---|---|
| **الف — ریشهٔ سند → `public/`** (پیشنهادی) | دامنهٔ افزوده یا زیردامنه که ریشهٔ سندش قابل تغییر است | `~/ritme/` (کل بسته) و ریشهٔ سند دامنه = `~/ritme/public` |
| **ب — بستهٔ `public_html`** | دامنهٔ اصلی حساب که ریشه‌اش ثابتِ `~/public_html` است | `~/ritme/` (برنامه، بیرون از وب) + `~/public_html/` (فقط فایل‌های عمومی و `index.php` اصلاح‌شده) |
| **ج — همه در `public_html`** (اضطراری) | وقتی هیچ‌کدام از دو حالت بالا ممکن نیست | کل بستهٔ «الف» داخل `~/public_html`؛ `.htaccess` ریشه همه‌چیز را به `public/` می‌فرستد و بقیه را نمی‌دهد |

در «ب» و «ج» هم کد برنامه از وب در دسترس نیست (`.env`، `vendor`، `storage`، `composer.json` … پاسخ ۴۰۳/۴۰۴ می‌دهند)،
اما «الف» و «ب» امن‌ترند چون برنامه اصلاً زیر ریشهٔ وب نیست.

---

## ۳. ساخت پایگاه داده

1. cPanel → **MySQL® Database Wizard** (یا MySQL® Databases).
2. پایگاه داده بسازید؛ cPanel پیشوند حساب را اضافه می‌کند: مثلاً `user_ritme`.
3. کاربر بسازید (`user_ritme`) با گذرواژهٔ قوی (از Password Generator).
4. کاربر را با **ALL PRIVILEGES** به همان پایگاه داده اضافه کنید (فقط همین پایگاه).

`[تصویر: Database Wizard — مرحلهٔ ساخت پایگاه داده]`
`[تصویر: Database Wizard — دادن ALL PRIVILEGES]`

---

## ۴. نسخه و تنظیمات PHP

1. cPanel → **MultiPHP Manager**: دامنه را روی **PHP 8.2** (یا 8.3) بگذارید. در هاست‌های CloudLinux: **Select PHP Version**.
2. در **Select PHP Version → Extensions** افزونه‌های جدول بخش ۰ را فعال کنید (`intl`، `exif`، `fileinfo`، `gd`، `pdo_mysql`، `zip`، `opcache`).
3. cPanel → **MultiPHP INI Editor** (یا Options در Select PHP Version):

| کلید | مقدار |
|---|---|
| `upload_max_filesize` | `16M` (کتابخانهٔ رسانه تا ۱۵ مگابایت می‌پذیرد) |
| `post_max_size` | `20M` |
| `memory_limit` | `256M` (بهینه‌سازی تصویر) |
| `max_execution_time` | `60` |

`[تصویر: MultiPHP Manager — انتخاب PHP 8.2]`
`[تصویر: Select PHP Version — افزونه‌ها]`

> نسخهٔ PHP خط فرمان (Terminal و cron) ممکن است با نسخهٔ وب فرق کند. `php -v` را در Terminal ببینید؛ اگر قدیمی بود از
> مسیر کامل استفاده کنید، مثلاً `/opt/cpanel/ea-php82/root/usr/bin/php` (یا `/usr/local/bin/ea-php82`).

---

## ۵. آپلود و باز کردن بسته

cPanel → **File Manager** → Upload، سپس روی zip راست‌کلیک → **Extract**.

- **الف:** پوشهٔ `~/ritme` بسازید و zip معمولی را آنجا extract کنید. سپس در **Domains** ریشهٔ سند دامنه را
  `ritme/public` بگذارید.
- **ب:** zip با پسوند `-public_html` را در **پوشهٔ خانه (`~`)** extract کنید؛ دو پوشهٔ `ritme/` و `public_html/` پر
  می‌شوند. اگر پوشهٔ برنامه یا ریشهٔ سند نام دیگری دارد، هنگام ساخت `--app-dir=…` و `--doc-root=…` بدهید (یا بعداً
  `php artisan app:install --public-path=/home/<user>/<docroot>` را اجرا و مسیر داخل `<docroot>/index.php` را اصلاح کنید).
- **ج:** zip معمولی را مستقیم داخل `~/public_html` extract کنید.

اگر هاست یک `index.html` پیش‌فرض در `public_html` گذاشته، آن را پاک کنید (`.htaccess` به هر حال `index.php` را ترجیح
می‌دهد). فایل‌های مخفی (`.htaccess`، `.env.cpanel.example`) را با **Settings → Show Hidden Files** در File Manager ببینید.

`[تصویر: File Manager — Extract]`
`[تصویر: Domains — تغییر Document Root به ritme/public]`

---

## ۶. فایل `.env`

نصب‌کننده اگر `.env` نباشد آن را از `.env.cpanel.example` می‌سازد، `APP_KEY` تولید می‌کند و متوقف می‌شود تا مقادیر را
پر کنید. یا خودتان کپی کنید و ویرایش کنید (File Manager → Edit):

| کلید | توضیح |
|---|---|
| `APP_URL` | آدرس اصلی سایت؛ تا صدور گواهی SSL می‌تواند `http://` باشد، بعد از آن `https://` |
| `APP_DEBUG` | **همیشه `false`** — نصب و ارتقا با `true` در production اجرا نمی‌شوند |
| `APP_CANONICAL_REDIRECT` | پس از فعال شدن SSL: `true` (http و www/non-www با یک ۳۰۱ به آدرس اصلی) |
| `DB_HOST` / `DB_DATABASE` / `DB_USERNAME` / `DB_PASSWORD` | معمولاً `localhost` و نام‌های پیشونددار بخش ۳ |
| `CACHE_STORE` | `file` (اگر سهمیهٔ inode تنگ است: `database`) |
| `SESSION_DRIVER` / `QUEUE_CONNECTION` | `file` / `database` |
| `MEDIA_PUBLIC_ROOT` | خالی بماند؛ فقط اگر پوشهٔ عمومی جای غیرعادی است مسیر مطلق `…/media` |
| `MAIL_*` | SMTP خود هاست (Email Accounts → Connect Devices): `mail.<domain>`، پورت ۴۶۵ (`smtps`) یا ۵۸۷ |
| `ADMIN_MFA_ROLES` | `super-admin,shop-manager,directory-manager,support` |
| `TRUSTED_PROXIES` | روی cPanel ساده خالی؛ فقط اگر CDN/پراکسی جلوی سایت است بازه‌های آن |

سطح دسترسی `.env` را **600** بگذارید (File Manager → Permissions).

---

## ۷. نصب

cPanel → **Terminal**:

```bash
cd ~/ritme                 # چیدمان ج: cd ~/public_html
php artisan app:install
```

نصب‌کننده به ترتیب: نسخهٔ PHP و افزونه‌ها و پوشه‌های قابل نوشتن را بررسی می‌کند (جدول ok/warn/FAIL)، پوشه‌های
`storage` و `<public>/media` را می‌سازد، در صورت نبود `APP_KEY` آن را تولید می‌کند، اتصال پایگاه داده را می‌آزماید،
migrationها را اجرا می‌کند، داده‌های پیش‌فرض production (تنظیمات، FAQ، نقش‌های مدیریت) را می‌نشاند، پیشنهاد ساخت
اولین مدیر را می‌دهد، کش‌ها را می‌سازد (`optimize` + `filament:optimize`)، نسخهٔ همهٔ فضاهای کش را بالا می‌برد، نقشهٔ سایت
را گرم می‌کند و **خط cron** را چاپ می‌کند.

گزینه‌ها: `--check` (فقط بررسی)، `--public-path=<پوشه>` (چیدمان ب با نام غیرپیش‌فرض)، `--no-admin`، `--skip-checks`،
`--force` (اجرای دوباره روی سایت نصب‌شده؛ برای نسخهٔ جدید از `app:upgrade` استفاده کنید).

اولین مدیر (اگر در نصب نساختید):

```bash
php artisan admin:create --name="مدیر" --email=you@example.com --role=super-admin
```

گذرواژه همیشه به‌صورت مخفی پرسیده می‌شود (حداقل ۱۲ نویسه، حرف و عدد). مدیر کل در اولین ورود باید برنامهٔ
Authenticator (TOTP) را فعال کند. پنل: `https://<domain>/admin` (مسیر با `ADMIN_PATH` قابل تغییر است).

`[تصویر: Terminal — خروجی app:install]`

### نصب بدون Terminal

یک cron **موقت** (Cron Jobs → Once Per Minute) بسازید، یک دقیقه صبر کنید و بعد **حذفش کنید**:

```
cd /home/<user>/ritme && /usr/local/bin/php artisan app:install --no-interaction > storage/logs/install.log 2>&1
```

خروجی را در `storage/logs/install.log` ببینید. ساخت مدیر (`admin:create`) گذرواژه را تعاملی می‌پرسد، پس به Terminal/SSH
نیاز دارد — از پشتیبانی هاست فعال‌سازی Terminal را بخواهید.

---

## ۸. Cron (زمان‌بند + صف)

cPanel → **Cron Jobs** → Common Settings: **Once Per Minute (`* * * * *`)**، فرمان (همان که نصب‌کننده چاپ کرد):

```
/usr/local/bin/php /home/<user>/ritme/artisan schedule:run >> /dev/null 2>&1
```

همین یک خط اجرا می‌کند: کارگر صف پایگاه‌داده هر دقیقه (`queue:work --stop-when-empty --max-time=55` در پس‌زمینه،
بدون هم‌پوشانی) برای اعلان‌ها، بهینه‌سازی تصاویر، IndexNow و ممیزی SEO؛ انتشار مطالب زمان‌بندی‌شده (هر دقیقه)؛
ذخیرهٔ شمارنده‌های بازدید و آمار ریدایرکت (هر ۵ دقیقه)؛ گرم کردن نقشهٔ سایت (هر ساعت)؛ پاک‌سازی لاگ ۴۰۴ (روزانه)؛
پاک‌سازی کارهای ناموفق قدیمی و لاگ فعالیت (روزانه)؛ ممیزی SEO هفتگی. فهرست کامل: `php artisan schedule:list`.

`[تصویر: Cron Jobs — Once Per Minute]`

> اگر ایمیل‌های cron می‌رسد، بخش Cron Email را خالی کنید یا `>> /dev/null 2>&1` را نگه دارید.

---

## ۹. SSL و آدرس نهایی

1. cPanel → **SSL/TLS Status** → **Run AutoSSL** (برای دامنه و `www`). `.htaccess` مسیر `/.well-known` را باز می‌گذارد.
2. پس از صدور گواهی در `.env`: `APP_URL=https://<domain>` و `APP_CANONICAL_REDIRECT=true`.
3. `php artisan config:cache && php artisan cache:ns bump pages sitemap`

HSTS در production روی https خودکار فرستاده می‌شود (`SECURITY_HSTS_SUBDOMAINS=true` فقط وقتی همهٔ زیردامنه‌ها https هستند).

`[تصویر: SSL/TLS Status — Run AutoSSL]`

---

## ۱۰. بررسی پس از انتشار

```bash
php artisan app:install --check                  # همه ok (warnها را بخوانید)
php artisan schedule:list
curl -sI https://<domain>/ | head -20             # 200 + CSP، HSTS، X-Content-Type-Options، بدون X-Powered-By
curl -sI http://<domain>/                         # یک ۳۰۱ به https://<domain>/
for p in /.env /composer.json /vendor/autoload.php /storage/logs/laravel.log /media/x.php; do
  curl -s -o /dev/null -w "$p %{http_code}\n" https://<domain>$p; done   # همه 403 یا 404
```

بقیهٔ فهرست: `docs/SECURITY.md` §۵ (سطح دسترسی‌ها، نقش‌های MFA، پایگاه داده فقط localhost، `composer audit`).

---

## ۱۱. ارتقا به نسخهٔ جدید

1. روی سیستم توسعه بستهٔ جدید را با **همان چیدمان** بسازید.
2. (پیشنهادی) از پایگاه داده و `.env` و `public/media` پشتیبان بگیرید (بخش ۱۲).
3. پوشه‌های کد قدیمی را پاک کنید تا فایل حذف‌شده‌ای باقی نماند: `app/`، `bootstrap/` (به‌جز `bootstrap/public-path.php` در
   صورت تغییر دستی)، `config/`، `database/migrations/`، `lang/`، `resources/`، `routes/`، `vendor/`، `public/build/`،
   `public/js|css|fonts/filament`. **دست نزنید به:** `.env`، `storage/`، `public/media/` (یا `public_html/media/`).
4. zip جدید را مثل نصب اول همان‌جا extract کنید (بازنویسی فایل‌ها را تأیید کنید). بسته هیچ `.env`، `storage` پرشده یا
   `media` ندارد، پس داده‌ها حفظ می‌شوند.
5. در Terminal:

```bash
cd ~/ritme
php artisan app:upgrade
```

`app:upgrade`: بررسی پیش‌نیازها → حالت نگهداری با کلید عبور (نشانی bypass را چاپ می‌کند؛ با باز کردنش کوکی می‌گیرید و
سایت را می‌بینید) → `migrate --force` → تنظیمات و نقش‌های تازه (فقط درج موارد نبوده) → پاک‌سازی و ساخت دوبارهٔ
کش‌ها → بالا بردن نسخهٔ همهٔ فضاهای کش (صفحه‌های کش‌شده هش critical CSS نسخهٔ قبل را دارند) → گرم کردن نقشهٔ سایت →
`queue:restart` → خروج از حالت نگهداری. اگر مرحله‌ای شکست بخورد سایت در حالت نگهداری می‌ماند؛ علت را رفع و دوباره
اجرا کنید (یا `php artisan up`). گزینه‌ها: `--secret=…`، `--no-maintenance`، `--skip-checks`.

نسخهٔ نصب‌شده: فایل `release.json` در پوشهٔ برنامه (build id، commit، زمان ساخت).

---

## ۱۲. پشتیبان‌گیری

سایت خودش هر شب از پایگاه داده و `media` پشتیبان می‌گیرد (`backup:run` با cron؛ دانلود از «سیستم ← پشتیبان‌ها» در
پنل؛ جزئیات و آزمون بازیابی در `docs/GO-LIVE.md` بخش ۹). در کنار آن پشتیبان هاست را هم نگه دارید: cPanel → **Backup**
(یا **JetBackup**) — پایگاه داده (MySQL)، `.env`،
`public/media` (یا `public_html/media`) و `storage/app`. پیش از هر ارتقا یک «Download a MySQL Database Backup» بگیرید.
`APP_KEY` را جدا و امن نگه دارید (بدون آن رمز MFA مدیران و داده‌های رمزشده خوانده نمی‌شوند).

---

## ۱۳. عیب‌یابی

| نشانه | علت محتمل و راه‌حل |
|---|---|
| **خطای ۵۰۰** بدون صفحهٔ لاراول | معمولاً PHP قدیمی (MultiPHP را روی 8.2 بگذارید) یا دستور غیرمجاز در `.htaccess` (در `error_log` پوشه یا cPanel → **Errors** ببینید؛ اگر `Options` مجاز نیست، خطوط `Options` را در `public/.htaccess` کامنت کنید). |
| ۵۰۰ با صفحهٔ «خطا» سایت | `storage/logs/laravel-<تاریخ>.log` را بخوانید. اگر بعد از ارتقاست: `php artisan optimize:clear && php artisan app:upgrade`. |
| `Permission denied` / صفحهٔ سفید | پوشه‌ها `755`، فایل‌ها `644`، `.env` `600`؛ `storage/` و `bootstrap/cache/` و `media/` باید برای کاربر حساب قابل نوشتن باشند. هرگز `777`. `php artisan app:install --check` پوشه‌های مشکل‌دار را نشان می‌دهد. |
| `open_basedir restriction in effect` | مسیر برنامه (`/home/<user>/ritme`) باید داخل `open_basedir` باشد؛ در MultiPHP INI Editor یا از پشتیبانی هاست بخواهید آن را به `/home/<user>` گسترش دهد. |
| `Class "…" not found` / افزونهٔ ناموجود | افزونهٔ گفته‌شده را در Select PHP Version فعال کنید (`intl` برای پنل مدیریت، `exif` و `gd` برای تصاویر). یادتان باشد CLI و وب ممکن است INI جدا داشته باشند. |
| `Vite manifest not found` یا صفحه بدون استایل | چیدمان اشتباه: ریشهٔ سند باید `public/` باشد (الف) یا پوشهٔ `public_html` بستهٔ «ب»؛ در «ب» مسیر داخل `public_html/index.php` و `ritme/bootstrap/public-path.php` را بررسی کنید. |
| همهٔ صفحه‌ها ۴۰۳ | در چیدمان «ج» یعنی `mod_rewrite` فعال نیست (از هاست بخواهید). |
| تصویرها بهینه نمی‌شوند / ایمیل‌ها نمی‌روند | cron اجرا نمی‌شود: مسیر PHP در خط cron، `proc_open` در `disable_functions`، و `php artisan queue:failed` را بررسی کنید. |
| آپلود تصویر بزرگ رد می‌شود | `upload_max_filesize` و `post_max_size` وب (نه فقط CLI) را بالا ببرید (بخش ۴). |
| سایت در حالت نگهداری مانده | `php artisan up` |
| ایمیل ارسال نمی‌شود | `MAIL_SCHEME=smtps` با پورت ۴۶۵ یا `MAIL_SCHEME=null` با ۵۸۷؛ نام کاربری = آدرس کامل صندوق؛ سپس `php artisan config:cache`. |
| بعد از تغییر `.env` اثری نیست | کش پیکربندی: `php artisan config:cache` (و در صورت تغییر آدرس: `php artisan cache:ns bump pages sitemap`). |

---

## پیوست — ساختار بسته

```
چیدمان الف / ج (ritme-site-<build-id>.zip)       چیدمان ب (ritme-site-<build-id>-public_html.zip)
.htaccess              ← فقط برای ج                ritme/            ← برنامه (بیرون از وب)
.env.cpanel.example                                  bootstrap/public-path.php → ../public_html
app/ bootstrap/ config/ database/ lang/              … مثل ستون چپ بدون public/
resources/{views,svg}/ routes/ storage/ vendor/    public_html/      ← ریشهٔ سند
public/  ← build/ (+critical/)، sw.js، media/،       index.php (مسیر ../ritme)، .htaccess، build/، sw.js،
           js|css|fonts/filament، icons/              media/، js|css|fonts/filament، icons/
release.json  artisan  composer.json/lock
```

فایل‌های ساخت: `deploy/build-cpanel.sh`، الگوها در `deploy/cpanel/` (`root.htaccess`، `index.public_html.php`،
`public-path.php`)، فرمان‌ها در `app/Console/Commands/AppInstall.php` و `AppUpgrade.php`، زمان‌بند در
`routes/console.php`.
