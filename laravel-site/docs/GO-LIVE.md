# فهرست کنترل انتشار سایت ریتمی (L10-02)

این سند آخرین گام پیش از روشن کردن دامنهٔ واقعی است: چه چیزهایی را **مالک سایت** باید تصمیم بگیرد یا پر کند، بسته
چطور ساخته و بررسی می‌شود، DNS و SSL، کنسول‌های جست‌وجو، نقشهٔ ۳۰۱ از سایت قبلی، پشتیبان‌گیری، پایش و بازگشت به نسخهٔ
قبل.

> **هیچ کدام از کارهای بیرونی این سند خودکار انجام نمی‌شود.** آپلود روی هاست، تغییر DNS، ثبت در Google Search Console
> و Bing Webmaster، ثبت سرویس پایش بیرونی و هر push یا deploy فقط با تصمیم و دست خود مالک انجام می‌شود. کد و ابزارها
> فقط آماده‌اند.

اسناد مرتبط: نصب و ارتقا روی cPanel — `docs/DEPLOY-CPANEL.md` · امنیت (§۵ فهرست کنترل) — `docs/SECURITY.md` ·
عملکرد — `docs/PERFORMANCE.md` · نتیجهٔ Lighthouse — `docs/qa/perf/README.md` · دسترس‌پذیری — `docs/qa/a11y/README.md` ·
وفاداری صفحه‌ها — `docs/qa/L3/README.md`.

---

## ترتیب کلی

| # | مرحله | چه کسی | بخش |
|---|---|---|---|
| ۱ | محتوا و تصمیم‌های باز را نهایی کنید | مالک | §۱ |
| ۲ | بسته را از یک commit مشخص بسازید و بررسی‌های محلی را سبز کنید | توسعه | §۲ |
| ۳ | هاست، پایگاه داده، `.env`، نصب، cron | مالک (با راهنمای DEPLOY-CPANEL) | §۳ |
| ۴ | AutoSSL، سپس DNS دامنهٔ اصلی | مالک | §۴ |
| ۵ | تنظیمات پنل مدیریت (لینک‌ها، OG، فروشگاه…) و محتوای واقعی | مالک / ویراستار | §۵ |
| ۶ | `app:doctor` سبز، `seo:audit`، Lighthouse/GTmetrix، بررسی امنیتی | مالک + توسعه | §۶ |
| ۷ | نقشهٔ ۳۰۱ از سایت قبلی + آزمون آن | مالک + سئو | §۷ |
| ۸ | Search Console و Bing Webmaster، ارسال نقشهٔ سایت | مالک | §۸ |
| ۹ | پشتیبان، آزمون بازیابی، نسخهٔ بیرون از هاست | مالک | §۹ |
| ۱۰ | پایش: هشدار ایمیلی، پایش `/up`، بازبینی‌های دوره‌ای | مالک | §۱۰ |

---

## ۱. پیش از انتشار — محتوا و تصمیم‌های مالک

این‌ها در کد قابل حدس نیستند؛ تا پر نشوند یا بخشی از صفحه پنهان می‌ماند یا متن جای‌نگهدار (`[...]`) دیده می‌شود.

### ۱.۱ تنظیمات پنل (پس از نصب، در `/admin`)

- [ ] **لینک‌های اپ** (`تنظیمات ← لینک‌های اپ`، `/admin/settings/app-links`): بازار، مایکت، Google Play، App Store،
      نسخهٔ وب. تا خالی است دکمه‌های «ورود»/«دانلود اپ» سربرگ، نشان‌های فروشگاه در فوتر و بخش دعوت به اپ، و QR پنهان‌اند
      (تصمیم L1-02) — صفحه‌ها کوتاه‌تر از طرح دیده می‌شوند.
- [ ] **شبکه‌های اجتماعی** (`/admin/settings/social`): اینستاگرام، تلگرام، لینکدین.
- [ ] **تصویر OG پیش‌فرض** (`/admin/settings/seo` → «تصویر اشتراک پیش‌فرض (Open Graph)»، ۱۲۰۰×۶۳۰): هشدار og:image را از
      همهٔ صفحه‌ها برمی‌دارد.
- [ ] **سازمان** (`/admin/settings/organization`) و **تماس** (`/admin/settings/contact`): نام حقوقی، آدرس، تلفن،
      `support_email` و `data_protection_email` واقعی (جای‌نگهدارها در JSON-LD حذف می‌شوند ولی در متن صفحه‌ها دیده می‌شوند).
- [ ] **فروشگاه** (`/admin/settings/shop`): هزینهٔ ارسال، آستانهٔ ارسال رایگان، سقف پرداخت در محل (COD)، آستانهٔ کمبود
      موجودی. خالی = «هزینهٔ ارسال هنگام تماس اعلام می‌شود».
- [ ] **PWA** (`/admin/settings/pwa`): یک بار ذخیره کنید تا نام اپ نصب‌شده درست ثبت شود.
- [ ] **کدهای تأیید مالکیت** (`/admin/seo/indexing` → تأیید مالکیت): فقط مقدار `content` متای Google و Bing (بخش §۸).

### ۱.۲ متن‌های حقوقی و محصول (فهرست L3-08 و کارهای بعدی)

- [ ] **حریم خصوصی و قوانین** (`/privacy`، `/terms`، `/admin/settings/legal`): نام و شمارهٔ ثبت و نشانی شرکت؛ سن و رضایت
      والدین؛ فهرست داده‌های اپ و مدت نگهداری؛ محل میزبانی/سرور؛ ارائه‌دهندهٔ پیامک و ایمیل؛ روند درخواست‌های قانونی؛
      بازهٔ پاک شدن از پشتیبان‌ها (پیشنهاد مطابق §۹: حداکثر ۶ ماه)؛ مدت نگهداری پیام‌ها و لاگ سرور (لاگ‌ها ۱۴ روز)؛ مهلت
      پاسخ؛ قانون حاکم؛ شرایط تعلیق حساب؛ مالک حقوق معنوی؛ سیاست بازنشر؛ سیاست بازپرداخت؛ ارسال و مرجوعی؛ مسئولیت
      ارائه‌دهندگان خدمات؛ حل اختلاف؛ تاریخ‌های «به‌روزرسانی».
- [ ] **دربارهٔ ما و مسئولیت اجتماعی**: داستان بنیان‌گذار، مأموریت، آمار، تیم، نام اعضای شورای علمی، لینک فرصت‌های شغلی؛
      برنامه‌ها، نقل‌قول بنیان‌گذار، دورهٔ گزارش شفافیت و لینک آن؛ متن سیاست بازبینی تحریریه.
- [ ] **پاسخ‌های FAQ** که از طرح با جای‌نگهدار وارد شده‌اند (مثلاً «[سیاست بازگشت وجه…]» در `/plus`) — `/admin` ← سؤالات متداول.
- [ ] **قیمت‌های Plus**: نمایش در سایت فعلاً خاموش است («قیمت در اپ»). اگر قیمت در سایت لازم است تصمیم بگیرید.
- [ ] **ارسال و مرجوعی فروشگاه**: متن «۷ روز بازگشت»، متن IRC، متن تحویل و کد تخفیف — تأیید یا اصلاح.
- [ ] **راهنمای مادر و کودک**: `[زمان بررسی]` و شرایط همکاری در فرم عضویت کسب‌وکار.
- [ ] **مجله**: نام متخصص بازبین (`[نام متخصص]`) و زندگی‌نامهٔ نویسنده‌ها (بدون آن صفحهٔ نویسنده noindex می‌ماند).
- [ ] ادعای «۸۶ مورد» صفحهٔ اصلی و نمونه‌تاریخ‌های ۱۴۰۵ در ابزارها — تأیید.
- [ ] **محتوای واقعی**: نصب production هیچ مطلب، محصول یا مکان نمونه‌ای ندارد (فقط تنظیمات، FAQ و نقش‌ها). فهرست‌های بی‌محتوا
      عمداً noindex هستند و با انتشار محتوای واقعی index می‌شوند.
- [ ] خط قرمزهای محتوا را در متن‌های تازه رعایت کنید: بدون ادعای تشخیص، بدون «حتماً/قطعاً/دقیق‌ترین/تضمینی»، بدون فشار فروش.

---

## ۲. ساخت بسته و بررسی‌های محلی (روی سیستم توسعه)

```bash
cd laravel-site
composer verify && php artisan seo:audit          # pint + phpstan + pest + build (critical CSS) + ممیزی سئو
composer audit && npm audit --omit=dev            # هر دو باید «بدون آسیب‌پذیری» باشند
node tools/lighthouse.mjs --all                   # ۵۶ اجرا؛ همه باید هدف‌ها را بزنند (docs/qa/perf)
bash deploy/build-cpanel.sh --ref=<commit> --layout=all
shasum -a 256 -c dist/ritme-site-<build-id>.zip.sha256
```

- [ ] بسته را **از commit مشخص** بسازید (`--ref`)، نه از درخت کاری (`--worktree` فقط برای آزمایش است).
- [ ] **آزمون واقعی PHP 8.2** (روی هاست یا در Docker) — تا امروز فقط با 8.4 اجرا شده و `config.platform.php=8.2` قفل
      وابستگی‌ها را تضمین می‌کند، نه رفتار اجرا را. یک راه:
      `docker run --rm -v "$PWD":/app -w /app php:8.2-cli php vendor/bin/pest --parallel`
      (یا روی هاست پس از نصب: `php -v` و `php artisan app:install --check`).
- [ ] نسخهٔ قبلی بسته را نگه دارید (برای بازگشت، §۱۱).

---

## ۳. هاست، `.env`، نصب و cron

گام‌به‌گام در `docs/DEPLOY-CPANEL.md` §۰–§۸. مقدارهای `.env` که پیش از انتشار باید **دقیقاً** بررسی شوند:

| کلید | مقدار production |
|---|---|
| `APP_ENV` / `APP_DEBUG` | `production` / **`false`** |
| `APP_KEY` | یک بار روی هاست ساخته شود؛ نسخه‌ای در مدیر گذرواژه (بدون آن رمز MFA مدیران و داده‌های رمزشده خوانده نمی‌شوند) |
| `APP_URL` | `https://<دامنهٔ اصلی>` (پس از صدور گواهی، §۴) |
| `APP_CANONICAL_REDIRECT` | `true` پس از فعال شدن SSL (http و www/بی‌www با یک ۳۰۱) |
| `ADMIN_MFA_ROLES` | خالی (پیش‌فرض) یا `super-admin,shop-manager,directory-manager,support` — هرگز کمتر |
| `ADMIN_PATH` / `ADMIN_SESSION_TIMEOUT` | پیشنهاد: مسیری غیر از `admin` / حداکثر `60` |
| `TRUSTED_PROXIES` | روی cPanel ساده **خالی**؛ فقط اگر CDN/پراکسی جلوی سایت است بازه‌های آن |
| `LOG_CHANNEL` / `LOG_LEVEL` / `LOG_DAILY_DAYS` | `daily` / `warning` / `14` (فایل روزانه، پاک شدن خودکار پس از ۱۴ روز) |
| `CACHE_STORE` / `SESSION_DRIVER` / `QUEUE_CONNECTION` | `file` / `file` / `database` |
| `MAIL_*` | SMTP خود هاست؛ `MAIL_FROM_ADDRESS` واقعی (نه example.com) |
| `OPS_ALERT_EMAIL` | صندوقی که کسی واقعاً می‌خواند: کار صفِ شکست‌خورده و پشتیبانِ ناموفق/قدیمی به آن ایمیل می‌شود |
| `BACKUP_MAX_STORAGE_MB` | کمتر از فضای آزاد حساب (پیش‌فرض ۴۰۰۰) |
| `BACKUP_ARCHIVE_PASSWORD` | اختیاری؛ اگر گذاشتید کنار `APP_KEY` نگه دارید |

- [ ] `.env` با دسترسی `600`؛ پوشه‌ها `755`، فایل‌ها `644`؛ هیچ `777`.
- [ ] `php artisan app:install` (اولین بار) یا `php artisan app:upgrade` (نسخه‌های بعد).
- [ ] اولین مدیر کل: `php artisan admin:create --role=super-admin` و فعال‌سازی Authenticator در اولین ورود؛ هیچ کاربر پیش‌فرضی نباشد.
- [ ] **cron یک‌دقیقه‌ای** (`* * * * * /usr/local/bin/php /home/<user>/ritme/artisan schedule:run >> /dev/null 2>&1`)
      و پس از دو دقیقه: `php artisan schedule:list`. همین cron صف، انتشار زمان‌بندی‌شده، گرم کردن نقشهٔ سایت و پشتیبان شبانه را اجرا می‌کند.
- [ ] باینری `mysqldump` روی هاست هست (`which mysqldump`) — پشتیبان پایگاه داده به آن نیاز دارد (روی cPanel معمولاً هست).

---

## ۴. SSL و DNS

1. [ ] **پیش از جابه‌جایی**، TTL رکوردهای فعلی دامنه را ۲۴ ساعت زودتر به ۳۰۰ ثانیه کم کنید (اگر سایت قبلی روشن است).
2. [ ] سایت را ابتدا روی زیردامنه یا دامنهٔ موقت هاست نصب و بررسی کنید (`APP_URL` همان آدرس، `APP_CANONICAL_REDIRECT=false`).
3. [ ] در پنل DNS: رکورد `A` دامنهٔ اصلی → IP هاست؛ `www` → `CNAME` به دامنهٔ اصلی (یا `A` همان IP). رکوردهای ایمیل
      (`MX`، `SPF`، `DKIM`، `DMARC`) را از cPanel ← Email Deliverability بردارید و ثبت کنید تا ایمیل‌های سایت اسپم نشوند.
4. [ ] پس از انتشار DNS: cPanel ← **SSL/TLS Status** ← Run AutoSSL برای دامنه و `www`.
5. [ ] در `.env`: `APP_URL=https://<دامنه>`، `APP_CANONICAL_REDIRECT=true`، سپس:
      ```bash
      php artisan config:cache
      php artisan cache:ns bump pages sitemap    # صفحه‌ها و نقشهٔ سایت با آدرس تازه دوباره ساخته شوند
      ```
6. [ ] `curl -sI http://<دامنه>/` و `curl -sI https://www.<دامنه>/` → یک ۳۰۱ به `https://<دامنه>/`.
7. [ ] HSTS خودکار فرستاده می‌شود؛ `SECURITY_HSTS_SUBDOMAINS=true` فقط وقتی همهٔ زیردامنه‌ها https دارند.

---

## ۵. پس از نصب، در پنل

- [ ] همهٔ موارد §۱.۱ را پر کنید؛ هر ذخیره کش مربوط را خودکار تازه می‌کند.
- [ ] هر مدیر با نقش مناسب (کمترین دسترسی لازم) ساخته شود؛ نقش‌های دارای داده‌های شخصی MFA اجباری دارند.
- [ ] **پس از هر deploy**: `php artisan cache:ns bump pages sitemap` (`app:upgrade` خودش همهٔ فضاها را بالا می‌برد) — صفحه‌های
      کش‌شده هش critical CSS نسخهٔ قبل را دارند و CSP جدید آن را نمی‌پذیرد.

---

## ۶. بررسی نهایی روی production

### ۶.۱ `php artisan app:doctor`

روی هاست، پس از نصب و هر ارتقا (و هر وقت چیزی عجیب بود) اجرا کنید. هر سطر `ok` / `warn` / `FAIL` و یک توضیح دارد؛
هر `FAIL` یعنی کد خروج ۱.

| بررسی | معنی و رفع |
|---|---|
| `APP_ENV=production`، `APP_DEBUG off`، `APP_KEY set and valid`، `APP_URL https` | مقدارهای §۳ |
| `APP_CANONICAL_REDIRECT on`، `Not in maintenance mode` (warn) | §۴ / `php artisan up` |
| `Database + migrations` | اتصال + نبود migration در انتظار (`app:upgrade`) |
| `Cache store`، `Session driver` | file/database؛ نوشتن و خواندن کش آزموده می‌شود |
| `Queue connection`، `Queue worker ran (≤ 5 min)`، `Queue backlog (≤ 5 min)` | کارگر صف که cron هر دقیقه اجرا می‌کند پس از هر اجرای موفق «ضربان» می‌نویسد (`storage/framework/cache/queue-heartbeat`)؛ نبودِ ضربان تازه یا کاری که بیش از ۵ دقیقه منتظر مانده = cron یا `proc_open` کار نمی‌کند |
| `Failed jobs (24 h)` (warn) | `php artisan queue:failed`، پس از رفع: `queue:retry all` |
| `Writable paths` | storage، bootstrap/cache، media، pending، پوشهٔ موقت پشتیبان |
| `Optimize caches present` | `php artisan optimize && php artisan filament:optimize` |
| `Log rotation + level` (warn) | `LOG_CHANNEL=daily`، `LOG_LEVEL=warning` |
| `Build assets`، `Critical CSS manifest matches build` | بسته ناقص یا قاطی دو نسخه؛ بستهٔ درست را دوباره extract کنید |
| `Health route /up`، `robots.txt (production rules)`، `sitemap.xml reachable`، `Home page renders` | درون‌برنامه‌ای (بدون شبکه) درخواست می‌شوند؛ آدرس‌های نقشهٔ سایت باید زیر `APP_URL` باشند (وگرنه `cache:ns bump sitemap`) |
| `Backup recent (local, ≤ 1 d)` | پشتیبانی جوان‌تر از یک روز؛ اولین بار: `php artisan backup:run` |
| `Mail configured`، `Operator alerts (OPS_ALERT_EMAIL)` (warn) | SMTP واقعی و نشانی هشدار |
| `ADMIN_MFA_ROLES covers PII roles` | §۳ |

آزمون ارسال ایمیل: `php artisan app:doctor --mail=you@<domain>` (یک ایمیل آزمایشی می‌فرستد؛ پوشهٔ اسپم را هم ببینید).
آستانهٔ صف: `--minutes=10`.

نمونهٔ اجرای سبز روی محیط شبه‌production محلی (۵ اکتبر ۲۰۲۶، با `APP_ENV=production`، کش‌های optimize، یک پشتیبان
و یک اجرای کارگر صف از زمان‌بند): `Healthy — 26 ok, 0 warn, 0 FAIL`. در همان اجرا، پیش از `cache:ns bump sitemap`،
بررسی نقشهٔ سایت و robots به‌درستی FAIL داد چون آدرس‌های کش‌شده هنوز روی آدرس توسعه بودند — همان چیزی که گام ۵ بخش §۴ رفع می‌کند.

### ۶.۲ سئو

- [ ] روی هاست: `php artisan seo:audit` — صفر خطا (هشدارهای باقی‌مانده را بخوانید؛ `--strict` هشدارها را هم خطا می‌شمارد).
      نتیجه در پنل: `/admin/seo/audit`؛ ممیزی هفتگی خودکار (شنبه ۰۴:۲۰).
- [ ] `https://<دامنه>/robots.txt` قانون‌های production و خط `Sitemap:` دارد؛ `https://<دامنه>/sitemap.xml` باز می‌شود
      و همهٔ `<loc>`ها با `https://<دامنه>/` شروع می‌شوند.
- [ ] نمونه‌صفحه‌ها: یک `<h1>`، canonical درست، og:image (پس از §۱.۱)، JSON-LD بدون جای‌نگهدار.

### ۶.۳ عملکرد

- [ ] **PageSpeed Insights** (موبایل و دسکتاپ) و **GTmetrix** (حساب خود مالک) روی همان ۱۴ قالب `docs/qa/perf/README.md`:
      Performance ≥ ۹۵ موبایل، SEO/Best Practices ۱۰۰، A11y ≥ ۹۵، CLS < ۰٫۱، LCP < ۲٫۵ ثانیه. نتیجه (Grade / LCP / CLS)
      را در `docs/qa/perf/README.md` ثبت کنید.
- [ ] هدرها: `curl -sI https://<دامنه>/ | grep -iE 'content-encoding|cache-control|content-security|strict-transport'`.

### ۶.۴ امنیت

`docs/SECURITY.md` §۵ را کامل تیک بزنید؛ مهم‌ترین‌ها:

```bash
curl -sI https://<دامنه>/ | head -25        # CSP بدون unsafe-inline، HSTS، nosniff، X-Frame-Options، بدون X-Powered-By
for p in /.env /composer.json /vendor/autoload.php /storage/logs/laravel.log /media/x.php /release.json; do
  curl -s -o /dev/null -w "$p %{http_code}\n" https://<دامنه>$p; done   # همه 403 یا 404
```

### ۶.۵ دسترس‌پذیری (دستی)

- [ ] یک دور **صفحه‌خوان** با صدای فارسی: VoiceOver (iOS/macOS) و TalkBack (اندروید) روی صفحهٔ اصلی، یک مقاله، فروشگاه
      ← سبد ← تسویه، فرم تماس و فرم نوبت. بررسی‌های خودکار (axe صفر خطا، Lighthouse A11y ۱۰۰) کیفیت اعلام‌ها را نمی‌سنجند.
- [ ] وقتی اولین عکس مکان/محصول بارگذاری شد: `node docs/qa/a11y/a11y-check.mjs --only place` (لایت‌باکس گالری).
- [ ] صفحه‌های تأیید نوبت و سفارش را با اولین نوبت/سفارش واقعی یک بار با صفحه‌کلید و صفحه‌خوان ببینید.

---

## ۷. نقشهٔ ۳۰۱ از سایت قبلی

نشانی‌های قدیمی `*.html` و ساختار وردپرسی (`/cycle/`، `/sitemap_index.xml`، `/wp-sitemap.xml`، مسیرهای
`/wp-content/uploads/ritme/*`) خودکار ۳۰۱ می‌شوند. بقیه:

1. [ ] فهرست نشانی‌های سایت قبلی را جمع کنید: نقشهٔ سایت قدیمی، گزارش «Pages» در Search Console (Export)، و صفحه‌های
       پربازدید آمار قبلی.
2. [ ] یک CSV با ستون‌های `from,to,code,regex,note` بسازید (فقط `from` لازم است؛ `code` پیش‌فرض ۳۰۱؛ `to` خالی = ۴۱۰).
       نمونه:
       ```
       from,to,code,regex,note
       /old-article-slug/,/blog/period-pain,301,0,مقالهٔ قدیمی
       /product/old-name/,/shop/product/new-name,301,0,
       /tag/(.*),/blog,301,1,برچسب‌های قدیمی
       ```
3. [ ] پنل ← سئو ← ریدایرکت‌ها ← **درون‌ریزی CSV** (تا ۵۰۰۰ سطر؛ سطرهای نامعتبر گزارش می‌شوند؛ زنجیره‌ها و حلقه‌ها خودکار
       رد یا کوتاه می‌شوند).
4. [ ] آزمون: همان فهرست (یک نشانی در هر سطر، مسیر نسبی) در `old-urls.txt`:
       ```bash
       while read -r p; do
         curl -s -o /dev/null -w "%{http_code} %{redirect_url}  $p\n" "https://<دامنه>$p"
       done < old-urls.txt | sort | tee redirect-check.txt
       grep -vE '^(301|200|410) ' redirect-check.txt     # باید خالی باشد
       ```
       هر ۳۰۱ باید **یک جهش** به صفحهٔ ۲۰۰ باشد: `curl -sIL <url> | grep -E '^HTTP|^location'`.
5. [ ] هفتهٔ اول هر روز پنل ← سئو ← **پایش ۴۰۴** را ببینید و از همان‌جا ریدایرکت بسازید.

---

## ۸. Google Search Console و Bing Webmaster

1. [ ] Search Console ← Add property ← **Domain** (رکورد TXT در DNS؛ همهٔ زیردامنه‌ها و http/https را پوشش می‌دهد) یا
       **URL prefix** با روش HTML tag: مقدار `content` را در `/admin/seo/indexing` ← تأیید مالکیت ← Google بگذارید
       (تگ `<meta>` بدون هیچ درخواست بیرونی در head همهٔ صفحه‌ها قرار می‌گیرد).
2. [ ] Sitemaps ← `https://<دامنه>/sitemap.xml` را ثبت کنید.
3. [ ] **URL Inspection** ← Request indexing برای: صفحهٔ اصلی، ۶ صفحهٔ مرحله (`/cycle`، `/ttc`، `/pregnancy`،
       `/postpartum`، `/menopause`، `/teen`)، `/blog`، `/shop`، `/directory`، `/plus`، و چند مقالهٔ مهم.
4. [ ] اگر دامنه عوض شده (نه فقط ساختار): Settings ← **Change of address** از property قدیمی.
5. [ ] Bing Webmaster ← Import from Google Search Console (ساده‌ترین)، یا تأیید با متای `msvalidate.01` در همان صفحهٔ
       تأیید مالکیت؛ نقشهٔ سایت را ثبت کنید. IndexNow (برای Bing/Yandex) در production خودکار است — کلید آن در
       `/admin/seo/indexing` ← IndexNow.
6. [ ] پس از ۳ تا ۷ روز: گزارش Pages (Indexed / Not indexed و دلیل‌ها)، Core Web Vitals، Enhancements (Breadcrumbs، Products،
       FAQ) و Manual actions را بررسی کنید.

---

## ۹. پشتیبان‌گیری

**چه چیزی، کجا، کی** (`config/backup.php`، spatie/laravel-backup):

| | |
|---|---|
| محتوا | dump پایگاه داده (`db-dumps/…sql`) + رسانه‌ها (`<public>/media`) + عکس‌های تأییدنشدهٔ عضویت (`storage/app/pending`) |
| بیرون از پشتیبان | کد (خود بستهٔ zip نسخه پشتیبان کد است) و `.env`/`APP_KEY` — این‌ها را جدا در مدیر گذرواژه نگه دارید |
| محل | `storage/app/private/ritme-site/<تاریخ>.zip` — بیرون از ریشهٔ وب |
| زمان (تهران) | پاک‌سازی ۰۱:۵۰، پشتیبان ۰۲:۱۰، بررسی سلامت ۰۸:۱۵ (همه از cron) |
| نگهداری | همه ۳ روز، روزانه ۱۴ روز، هفتگی ۸ هفته، ماهانه ۶ ماه، سالانه ۱ سال؛ جدیدترین هرگز پاک نمی‌شود؛ سقف حجم `BACKUP_MAX_STORAGE_MB` |
| هشدار | شکست پشتیبان/پاک‌سازی یا پشتیبانِ قدیمی‌تر از یک روز → ایمیل به `OPS_ALERT_EMAIL` (موفقیت ایمیل نمی‌شود) |
| دانلود | پنل ← سیستم ← **پشتیبان‌ها** (فقط مدیر کل؛ هر دانلود در گزارش فعالیت ثبت می‌شود) |

فرمان‌ها: `php artisan backup:run` (فوری)، `backup:list`، `backup:monitor`، `backup:clean`.

- [ ] اولین پشتیبان را همان روز نصب دستی بگیرید (`php artisan backup:run`) و در پنل ببینید.
- [ ] **نسخهٔ بیرون از هاست**: هفته‌ای یک بار آخرین zip را از پنل دانلود و در جای امن (رمزگذاری‌شده) نگه دارید. پشتیبان
      محلی با خراب شدن خود حساب هاست از بین می‌رود. (افزودن مقصد دور — FTP/SFTP/S3 — بعداً با یک دیسک در
      `config/filesystems.php` و `BACKUP_DISKS=local,<disk>` ممکن است.)
- [ ] پشتیبان خود cPanel/JetBackup را هم فعال نگه دارید (لایهٔ دوم).
- [ ] **آزمون بازیابی** یک بار پیش از انتشار، روی یک نصب آزمایشی (نه production):
      1. zip را باز کنید؛ پایگاه داده: phpMyAdmin ← پایگاه دادهٔ خالی ← Import ← فایل `db-dumps/*.sql`
         (یا `mysql -u <user> -p <db> < db-dumps/<file>.sql`).
      2. پوشهٔ `media` را به جای `<public>/media` و `pending` را به `storage/app/pending` کپی کنید.
      3. همان `APP_KEY` قبلی در `.env`؛ سپس `php artisan app:upgrade` و `php artisan app:doctor`.
      4. ورود مدیر با MFA، یک مقاله با تصویر و یک سفارش را باز کنید.
- [ ] پیش از هر ارتقا یک `php artisan backup:run` دستی.

---

## ۱۰. پایش

- [ ] **هشدار ایمیلی** (`OPS_ALERT_EMAIL`): کار صفی که پس از همهٔ تلاش‌ها شکست بخورد (حداکثر یک ایمیل برای هر نوع کار
      در هر ساعت، بدون دادهٔ شخصی؛ ایمیل و شماره‌ها در متن خطا پوشانده می‌شوند) + هشدارهای پشتیبان. با
      `php artisan app:doctor --mail=…` تحویل ایمیل را بیازمایید.
- [ ] **پایش در دسترس بودن** از بیرون (انتخاب و ثبت با مالک؛ مثلاً UptimeRobot یا سرویس مانیتور خود هاست): هر ۵ دقیقه
      `https://<دامنه>/up` (پاسخ ۲۰۰، بدون کوکی و بدون پایگاه داده‌ای سنگین) و صفحهٔ اصلی. این فقط درخواستی **به** سایت
      است و قانون «بدون درخواست بیرونی از صفحه‌های عمومی» را نقض نمی‌کند.
- [ ] **لاگ‌ها**: `storage/logs/laravel-YYYY-MM-DD.log` (روزانه، ۱۴ روز، سطح warning). خطاهای PHP پیش از لاراول:
      cPanel ← Errors یا `error_log` پوشه.
- [ ] **بازبینی دوره‌ای**:

| کی | چه |
|---|---|
| پس از هر deploy | `php artisan app:doctor`، `php artisan schedule:list`، یک بار باز کردن صفحهٔ اصلی و پنل |
| روزانه (هفتهٔ اول) | پایش ۴۰۴ در پنل، `queue:failed`، صندوق `OPS_ALERT_EMAIL`، Search Console ← Pages |
| هفتگی | گزارش ممیزی سئو (`/admin/seo/audit`)، دانلود پشتیبان بیرون از هاست، `app:doctor` |
| ماهانه | `composer audit` و `npm audit --omit=dev` روی commit جاری، Core Web Vitals در Search Console، فضای آزاد هاست در برابر `BACKUP_MAX_STORAGE_MB` |
| فصلی | آزمون بازیابی (§۹)، بازبینی فهرست مدیران و نقش‌ها، گزارش فعالیت |

---

## ۱۱. بازگشت (rollback)

1. `php artisan down --secret=<یک-رشته>` (یا اگر `app:upgrade` شکست خورد، سایت خودش در حالت نگهداری مانده است).
2. پوشه‌های کد را پاک و **بستهٔ نسخهٔ قبل** را مثل ارتقا extract کنید (`docs/DEPLOY-CPANEL.md` §۱۱).
3. اگر migration جدید داده را عوض کرده: پایگاه داده را از پشتیبانِ پیش از ارتقا بازیابی کنید (§۹).
4. `php artisan app:upgrade` → `php artisan app:doctor` → `php artisan up`.

---

## ۱۲. موارد باز غیرمسدودکننده (پس از انتشار)

- رسانه‌های `pending` هنوز در کتابخانهٔ رسانه دیده می‌شوند (فیلتر بعدی).
- ۴۴ پیکسل برای همهٔ هدف‌های لمسی دسکتاپ برآورده نیست (AA 2.2 یعنی ۲۴ پیکسل همه‌جا برآورده است).
- فونت‌های جایگزین اندروید/لینوکس از نظر متریک هم‌تراز نیستند؛ CLS واقعی اندروید را در Search Console ببینید.
- عنوان/توضیح `/offline` برای ممیزی کوتاه است (صفحهٔ noindex).
- پرداخت فقط در محل است؛ درگاه آنلاین تصمیم بعدی است.
