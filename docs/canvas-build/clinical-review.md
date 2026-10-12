# بازبینی بالینی — محتوای کانواس (یائسگی، IVF، از دست دادن بارداری و ...)

> این سند برای **متخصص زنان / ماما** است. هر مورد در کد یا seed که با «[needs clinical review]» علامت خورده، اینجا فهرست شده است. ستون «تصمیم بالینی» را پر کنید (تأیید / اصلاح به ... / حذف) و سند را برگردانید.

**یادداشت دربارهٔ منبع محتوا:** متن‌ها و مقادیر زیر **از محتوای بردهای طراحی (canvas boards) به‌عنوان نقطهٔ شروع** گرفته شده‌اند و توسط تیم محصول/مهندسی نوشته یا تکمیل شده‌اند؛ هیچ‌کدام هنوز تأیید بالینی نشده‌اند. عبارت‌های فارسی عیناً از کد نقل شده‌اند و ممکن است ناهماهنگی‌های جزئی نگارشی داشته باشند. هیچ مقداری در این سند اختراع نشده است.

پرچم `needs_review` ردیف‌های کاتالوگ را ادمین بعد از تأیید از پنل ادمین (گروه‌های کاتالوگ) برمی‌دارد؛ قواعد و آستانه‌های داخل کد پس از تصمیم شما توسط مهندسی تغییر می‌کنند.

خارج از دامنه: کاتالوگ‌های واکسن/نقاط عطف کودک، آزمایش‌ها (lab_markers)، تله‌مدیسین، بارداری پس از زایمان (EPDS) و قاعدهٔ انقباضات زایمان متعلق به کارهای bloom/قبلی‌اند و در این سند نیامده‌اند. ردیف‌های `needs_review = 0` (خدمات، دایرکتوری) بالینی نیستند.

## خلاصه (تعداد موارد)

| حوزه | ردیف کاتالوگ/چکاپ | قواعد و متن در کد | پیشنهاد QA | جمع |
|---|---|---|---|---|
| کف لگن (Pelvic floor) | 6 | 3 | 0 | 9 |
| پیشگیری از بارداری (Contraception) | 4 | 8 | 0 | 12 |
| آی‌وی‌اف (IVF) | 36 | 7 | 0 | 43 |
| از دست دادن بارداری (Loss) | 27 | 6 | 5 | 38 |
| یائسگی (Menopause) | 49 | 13 | 0 | 62 |
| بیماری‌ها و برنامه‌ها: درد/اندومتریوز، PMDD، خونریزی سنگین PBAC (Conditions) | 28 | 6 | 0 | 34 |
| نوجوان (Teen) | 17 | 4 | 5 | 26 |
| پرونده سلامت و اسناد (Record) | 0 | 5 | 0 | 5 |
| ثبت صوتی (Voice) | 0 | 4 | 0 | 4 |
| **جمع** | **167** | **56** | **10** | **233** |

شناسه‌ها: `-C` = قاعده/متن در کد، `-K` = ردیف کاتالوگ، `-Q` = پیشنهاد QA. پیشوندها: PF کف لگن، CT پیشگیری، IV آی‌وی‌اف، LS از دست دادن، MP یائسگی، CN بیماری‌ها، TN نوجوان، RC پرونده، VC صوتی.

## کف لگن (Pelvic floor)

برنامهٔ ۸ هفتهٔ کگل با چهار سطح، دفترچهٔ مثانه و هشدار علائم عفونت ادراری. لطفاً زمان‌بندی سطح‌ها (نگه‌داشتن/استراحت/تکرار/ست) و متن هشدارها را تأیید یا اصلاح کنید.

### الف) قواعد، آستانه‌ها و متن‌های داخل کد

| شناسه | چه چیزی باید بررسی شود | مقدار / متن فعلی (عین کد) | منبع | تصمیم بالینی |
|---|---|---|---|---|
| PF-C01 | طول برنامه و منطق انتخاب سطح: ۸ هفته؛ سطح = آخرین ردیف با week_from ≤ هفتهٔ جاری؛ مدت جلسه = ست × تکرار × (نگه‌داشتن + استراحت) | `WeeksTotal = 8`؛ هر دو هفته یک سطح (week_from = 1، 3، 5، 7)؛ ۱۰ تکرار × ۳ ست؛ docs: «weeks 1–2 → 3 s/3 s, 3–4 → 5 s/5 s, 5–6 → 8 s/8 s, 7–8 → 10 s/10 s, 10 reps × 3 sets» | `backend-go/internal/pelvic/program.go ; docs/canvas-build/pelvic.md:11` | |
| PF-C02 | متن معرفی برنامه و پایان برنامه | «روزی چند دقیقه تمرین کگل با راهنمای زمان‌دار؛ سطح تمرین هر دو هفته کمی سخت‌تر می‌شود.» / «۸ هفته را تمام کردی. می‌توانی با همین سطح ادامه بدهی.» | `frontend/messages/fa/pelvic.json (plan.join.body, plan.completed)` | |
| PF-C03 | گزینه‌های دفترچهٔ مثانه: نوع نشت، علائم عفونت ادراری، حداکثر بیدارشدن شبانه؛ هشدار UTI با ثبت هر علامت فعال می‌شود | Leaks: `none, cough, urgency, unexplained`؛ UTISymptoms: `burning, frequency, cloudy_odor`؛ `maxNightVoids = 20`؛ uti_alert = ثبت هر یک از علائم UTI | `backend-go/internal/pelvic/program.go ; backend-go/internal/pelvic/view.go` | |

### ب) ردیف‌های کاتالوگ (جدول catalog_items در seed)

همهٔ ردیف‌ها با `needs_review = 1` ذخیره شده‌اند. مقادیر زیر متن فارسی seed هستند (نسخهٔ انگلیسی در همان فایل است). ادمین بعد از تأیید می‌تواند پرچم را بردارد.

| شناسه | گروه / کد | چه چیزی باید بررسی شود | مقدار / متن فعلی | منبع | تصمیم بالینی |
|---|---|---|---|---|---|
| PF-K01 | `pelvic_levels` / `level_1` | هفتهٔ شروع سطح، مدت انقباض/استراحت (ثانیه)، تعداد تکرار و ست، و متن دستورالعمل | **عنوان:** «سطح ۱»<br>**متن:** «عضلاتی را منقبض کن که با آن جلوی ادرار را می‌گیری. شکم، باسن و ران‌ها شل بمانند و نفست را حبس نکن.»<br>**meta:** `{"week_from": 1, "hold_sec": 3, "rest_sec": 3, "reps": 10, "sets": 3}` | `backend-go/db/migrations/00010_pelvic_floor.sql:63` | |
| PF-K02 | `pelvic_levels` / `level_2` | هفتهٔ شروع سطح، مدت انقباض/استراحت (ثانیه)، تعداد تکرار و ست، و متن دستورالعمل | **عنوان:** «سطح ۲»<br>**متن:** «عضلاتی را منقبض کن که با آن جلوی ادرار را می‌گیری. شکم، باسن و ران‌ها شل بمانند و نفست را حبس نکن.»<br>**meta:** `{"week_from": 3, "hold_sec": 5, "rest_sec": 5, "reps": 10, "sets": 3}` | `backend-go/db/migrations/00010_pelvic_floor.sql:67` | |
| PF-K03 | `pelvic_levels` / `level_3` | هفتهٔ شروع سطح، مدت انقباض/استراحت (ثانیه)، تعداد تکرار و ست، و متن دستورالعمل | **عنوان:** «سطح ۳»<br>**متن:** «عضلاتی را منقبض کن که با آن جلوی ادرار را می‌گیری. شکم، باسن و ران‌ها شل بمانند و نفست را حبس نکن.»<br>**meta:** `{"week_from": 5, "hold_sec": 8, "rest_sec": 8, "reps": 10, "sets": 3}` | `backend-go/db/migrations/00010_pelvic_floor.sql:71` | |
| PF-K04 | `pelvic_levels` / `level_4` | هفتهٔ شروع سطح، مدت انقباض/استراحت (ثانیه)، تعداد تکرار و ست، و متن دستورالعمل | **عنوان:** «سطح ۴»<br>**متن:** «عضلاتی را منقبض کن که با آن جلوی ادرار را می‌گیری. شکم، باسن و ران‌ها شل بمانند و نفست را حبس نکن.»<br>**meta:** `{"week_from": 7, "hold_sec": 10, "rest_sec": 10, "reps": 10, "sets": 3}` | `backend-go/db/migrations/00010_pelvic_floor.sql:75` | |
| PF-K05 | `pelvic_alerts` / `uti_warning` | متن هشدار (علائم عفونت ادراری و ارجاع) یا جملهٔ مناسب‌بودن برنامه | **عنوان:** «اگر تب، لرز یا درد پهلو داری، زودتر به پزشک مراجعه کن.»<br>**meta:** `{"severity": "urgent"}` | `backend-go/db/migrations/00010_pelvic_floor.sql:79` | |
| PF-K06 | `pelvic_alerts` / `program_suitability` | متن هشدار (علائم عفونت ادراری و ارجاع) یا جملهٔ مناسب‌بودن برنامه | **عنوان:** «مناسب بعد از زایمان، یائسگی و هر وقت نشت ادرار داری. اگر درد لگن یا سنگینی داری، اول با پزشک یا فیزیوتراپ لگن مشورت کن.»<br>**meta:** `{"severity": "info"}` | `backend-go/db/migrations/00010_pelvic_floor.sql:83` | |


## پیشگیری از بارداری (Contraception)

روش پیشگیری کاربر، بستهٔ قرص، یادآورهای آی‌یودی/آمپول/کاشتنی و راهنمای قرص فراموش‌شده. مهم‌ترین مورد: قواعد قرص فراموش‌شده (گام‌ها و هفتهٔ بدون محافظت) و مقادیر پیش‌فرض عمر آی‌یودی.

### الف) قواعد، آستانه‌ها و متن‌های داخل کد

| شناسه | چه چیزی باید بررسی شود | مقدار / متن فعلی (عین کد) | منبع | تصمیم بالینی |
|---|---|---|---|---|
| CT-C01 | ساختار بستهٔ قرص: ترکیبی ۲۱+۷ (۷ روز بدون قرص)، ۲۸ (۲۱ فعال + ۷ دارونما)، ۲۴+۴ (۲۴ فعال + ۴ دارونما)؛ قرص فقط‌پروژسترون ۲۸ روز فعال بدون وقفه | `Pack21Plus7 = "21_7"`؛ `Pack28 → Active 21 + Placebo`؛ `Pack24Plus4 → Active 24 + Placebo`؛ `MethodProgestinPill → Pack{Length: 28, Active: 28}` | `backend-go/internal/contraception/pack.go` | |
| CT-C02 | طول عمر پیش‌فرض آی‌یودی (مقدار اولیهٔ انتخابگر) و بازهٔ مجاز | `copper_iud: 10`، `hormonal_iud: 5` سال؛ بازه ۱ تا ۱۲ سال؛ تاریخ تعویض = تاریخ گذاشتن + عمر دستگاه | `frontend/src/entities/contraception/model/pack.ts ; backend-go/internal/contraception/pack.go (MinIUDLifetime=1, MaxIUDLifetime=12)` | |
| CT-C03 | ویزیت کنترل آی‌یودی ۶ هفته پس از گذاشتن و یادآور ماهانهٔ بررسی نخ | `FollowupDays = 42`؛ یادآور ماهانه از «تاریخ گذاشتن + ۱ ماه»؛ «ماهانه، بعد از پریود»؛ «اگر نخ آی‌یودی را حس نکردی یا درد و تب داشتی، به پزشک مراجعه کن.» | `backend-go/internal/contraception/pack.go ; backend-go/internal/contraception/reminders.go ; frontend/messages/fa/contraception.json (other.iud.*)` | |
| CT-C04 | فاصلهٔ آمپول و زمان نمایش دکمهٔ «تزریق را زدم» | `InjectionDays = 84` («معمولاً هر ۱۲ هفته»)؛ `INJECTION_DONE_WINDOW = 21` روز پیش از تزریق بعدی؛ `INJECTION_WEEKS = 12` | `backend-go/internal/contraception/pack.go ; frontend/src/entities/contraception/model/missed.ts` | |
| CT-C05 | کاشتنی (ایمپلنت): بدون مدت پیش‌فرض؛ کاربر خودش تاریخ برداشتن/تعویض را می‌گذارد؛ متن خنثی برای آمپول/کاشتنی | «طبق نوع کاشتنی و نظر پزشک تنظیم کن» / «تاریخ‌ها را با پزشک یا ماما هماهنگ کن و هر وقت عوض شد، اینجا به‌روزش کن.» | `frontend/messages/fa/contraception.json (other.implant.replaceSub, other.note)` | |
| CT-C06 | یادآور تمام‌شدن بستهٔ قرص و حداکثر بستهٔ باقی‌مانده | `RefillDaysLead = 5` («یادآور خرید ۵ روز قبل از تمام شدن»)؛ `MaxPacksLeft = 24` | `backend-go/internal/contraception/pack.go` | |
| CT-C07 | ساعت یادآورها (ویزیت‌ها یک روز قبل یادآوری می‌شوند) | `ReminderHour = 9` (تهران)؛ `RemindBefore = care.DefaultRemindBefore` | `backend-go/internal/contraception/reminders.go` | |
| CT-C08 | متن‌های جایگزین راهنمای قرص جاافتاده وقتی کاتالوگ در دسترس نیست | «راهنما فعلاً در دسترس نیست» / «در صورت شک با پزشک یا داروساز صحبت کن.» | `frontend/messages/fa/contraception.json (missed.unavailable.*)` | |

### ب) ردیف‌های کاتالوگ (جدول catalog_items در seed)

همهٔ ردیف‌ها با `needs_review = 1` ذخیره شده‌اند. مقادیر زیر متن فارسی seed هستند (نسخهٔ انگلیسی در همان فایل است). ادمین بعد از تأیید می‌تواند پرچم را بردارد.

| شناسه | گروه / کد | چه چیزی باید بررسی شود | مقدار / متن فعلی | منبع | تصمیم بالینی |
|---|---|---|---|---|---|
| CT-K01 | `missed_pill_rules` / `combined_one` | قاعدهٔ قرص فراموش‌شده: روش‌های مشمول، تعداد قرص جاافتاده، گام‌ها، شدت و هفتهٔ بسته | **عنوان:** «۱ قرص (کمتر از ۴۸ ساعت دیر)»<br>**متن:** «این راهنمای عمومی قرص ترکیبی است. در صورت شک با پزشک یا داروساز صحبت کن.»<br>**meta:** `{"methods": ["combined_pill"], "missed": 1, "severity": "caution", "steps": ["قرص جاافتاده را همین حالا بخور، حتی اگر یعنی امروز دو قرص بخوری.", "بقیه قرص‌ها را طبق معمول ادامه بده."]}` | `backend-go/db/migrations/00017_contraception.sql:76` | |
| CT-K02 | `missed_pill_rules` / `combined_two_plus` | قاعدهٔ قرص فراموش‌شده: روش‌های مشمول، تعداد قرص جاافتاده، گام‌ها، شدت و هفتهٔ بسته | **عنوان:** «۲ قرص یا بیشتر»<br>**متن:** «این راهنمای عمومی قرص ترکیبی است. در صورت شک با پزشک یا داروساز صحبت کن.»<br>**meta:** `{"methods": ["combined_pill"], "missed": 2, "severity": "caution", "steps": ["آخرین قرص جاافتاده را همین حالا بخور، حتی اگر یعنی امروز دو قرص بخوری.", "بقیه قرص‌ها را طبق معمول ادامه بده.", "تا ۷ روز پشت سر هم قرص نخورده‌ای، از کاندوم استفاده کن.", "اگر در ۷ روز آخر قرص‌های فعال هستی، بعد از تمام شدنشان روزهای استراحت را حذف کن و بسته بعد را مستقیم شروع کن."]}` | `backend-go/db/migrations/00017_contraception.sql:81` | |
| CT-K03 | `missed_pill_rules` / `week1_unprotected` | قاعدهٔ قرص فراموش‌شده: روش‌های مشمول، تعداد قرص جاافتاده، گام‌ها، شدت و هفتهٔ بسته | **عنوان:** «اگر در هفته اول بسته رابطه محافظت‌نشده داشتی»<br>**متن:** «ممکن است به پیشگیری اضطراری نیاز داشته باشی. هر چه زودتر با پزشک یا داروساز مشورت کن.»<br>**meta:** `{"methods": ["combined_pill"], "severity": "urgent", "pack_week": 1}` | `backend-go/db/migrations/00017_contraception.sql:86` | |
| CT-K04 | `missed_pill_rules` / `progestin_note` | قاعدهٔ قرص فراموش‌شده: روش‌های مشمول، تعداد قرص جاافتاده، گام‌ها، شدت و هفتهٔ بسته | **عنوان:** «قرص تک‌هورمونی»<br>**متن:** «برای قرص تک‌هورمونی قواعد فرق دارد و به نوع قرص بستگی دارد؛ در صورت شک با پزشک یا داروساز صحبت کن.»<br>**meta:** `{"methods": ["progestin_pill"], "severity": "caution", "steps": ["قرص جاافتاده را به محض یادآوری بخور و قرص بعدی را سر ساعت همیشگی بخور.", "اگر بیشتر از زمان مجاز نوع قرصت دیر کردی، تا ۲ روز از کاندوم استفاده کن."]}` | `backend-go/db/migrations/00017_contraception.sql:91` | |


## آی‌وی‌اف (IVF)

مراحل چرخه، پروتکل‌ها، محل‌های تزریق، داروهای پیش‌فرض، راهنماها و علائم خطر، به‌علاوهٔ قواعد کد (آستانهٔ کمبود دارو، پیشنهاد توقف دارو بین مراحل، ثبت سونو و دو هفتهٔ انتظار).

### الف) قواعد، آستانه‌ها و متن‌های داخل کد

| شناسه | چه چیزی باید بررسی شود | مقدار / متن فعلی (عین کد) | منبع | تصمیم بالینی |
|---|---|---|---|---|
| IV-C01 | آستانهٔ «کم‌بودن موجودی دارو» | `LowSupplyDays = 3`: موجودی ≤ ۳ روز و پیش از پایان دارو تمام شود → «کم است» | `backend-go/internal/ivf/ivf.go ; frontend/src/entities/ivf/model/meds.ts` | |
| IV-C02 | ساعت نوبت آزمایش بتا (از beta_on ساخته می‌شود) | `BetaHour = 8` (تهران) | `backend-go/internal/ivf/ivf.go ; backend-go/internal/ivf/appointments.go` | |
| IV-C03 | پیشنهاد توقف داروها هنگام رفتن از آماده‌سازی/تحریک به مرحلهٔ جمع‌آوری یا بعد | نقش‌های `stimulation` و `suppression` (نه تریگر، نه حمایت لوتئال) | `frontend/src/entities/ivf/model/meds.ts (IVF_STIMULATION_ROLES) ; frontend/src/screens/ivf-cycle/model/cycle.ts (medsToStop)` | |
| IV-C04 | بازه‌های اندازهٔ فولیکول در ثبت سونو، حدود آندومتر و واحد استرادیول | Bins: `lt_10, 10_14, 15_17, 18_plus` (برچسب‌ها: «کمتر از ۱۰ میلی‌متر»، «۱۰ تا ۱۴»، «۱۵ تا ۱۷»، «۱۸ و بیشتر»)؛ آندومتر `min:0 max:40` mm؛ `MaxFollicles = 60`؛ E2 واحد `pg_ml` (پیش‌فرض) یا `pmol_l` | `backend-go/internal/ivf/ivf.go ; backend-go/internal/ivf/request.go ; frontend/messages/fa/ivf.json (scan.bins.*)` | |
| IV-C05 | یادداشت صفحهٔ سونو | «تفسیر و تصمیم درباره زمان تخمک‌کشی با پزشکت است. ما فقط کمک می‌کنیم همه‌چیز یک‌جا ثبت شود.» | `frontend/messages/fa/ivf.json (scan.note)` | |
| IV-C06 | متن‌های دو هفتهٔ انتظار (TWW) و نتیجه | «هر حسی داری طبیعی است. تست خانگی زودتر از موعد می‌تواند گمراه‌کننده باشد.» / مثبت: «داروهای حمایتی را تا وقتی پزشکت می‌گوید ادامه بده.» / منفی و لغو: «یادآور داروهای این دوره خاموش شد. پیش از قطع هر دارو با پزشکت هماهنگ کن.» | `frontend/messages/fa/ivf.json (tww.mood.note, tww.outcome.*.note)` | |
| IV-C07 | رفتار پس از نتیجه: یادآور داروها پس از نتیجهٔ مثبت ادامه دارد؛ نتیجهٔ منفی پیوند «از دست دادن» را فقط با انواع از دست دادن بارداری نشان می‌دهد | `NextSteps`: مثبت → `pregnancy_setup`؛ منفی → `loss`, `new_cycle`؛ لغو → `new_cycle` (پرسش: افزودن «انتقال ناموفق»؟) | `backend-go/internal/ivf/ivf.go (NextSteps) ; docs/qa/canvas/ivf.md` | |

### ب) ردیف‌های کاتالوگ (جدول catalog_items در seed)

همهٔ ردیف‌ها با `needs_review = 1` ذخیره شده‌اند. مقادیر زیر متن فارسی seed هستند (نسخهٔ انگلیسی در همان فایل است). ادمین بعد از تأیید می‌تواند پرچم را بردارد.

| شناسه | گروه / کد | چه چیزی باید بررسی شود | مقدار / متن فعلی | منبع | تصمیم بالینی |
|---|---|---|---|---|---|
| IV-K01 | `ivf_stages` / `prep` | مراحل چرخه و توضیح هر مرحله | **عنوان:** «آماده‌سازی»<br>**متن:** «آزمایش‌ها و سونوی پایه»<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:162` | |
| IV-K02 | `ivf_stages` / `stim` | مراحل چرخه و توضیح هر مرحله | **عنوان:** «تحریک تخمک‌گذاری»<br>**متن:** «تزریق روزانه · حدود ۱۰ تا ۱۲ روز»<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:167` | |
| IV-K03 | `ivf_stages` / `retrieval` | مراحل چرخه و توضیح هر مرحله | **عنوان:** «تخمک‌کشی»<br>**متن:** «حدود ۳۶ ساعت بعد از تزریق تریگر، طبق برنامه کلینیک»<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:172` | |
| IV-K04 | `ivf_stages` / `transfer` | مراحل چرخه و توضیح هر مرحله | **عنوان:** «انتقال جنین»<br>**متن:** «تاریخ را کلینیک بر اساس رشد جنین تعیین می‌کند»<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:177` | |
| IV-K05 | `ivf_stages` / `tww` | مراحل چرخه و توضیح هر مرحله | **عنوان:** «انتظار دو هفته‌ای»<br>**متن:** «داروها را طبق نسخه ادامه بده»<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:182` | |
| IV-K06 | `ivf_stages` / `test` | مراحل چرخه و توضیح هر مرحله | **عنوان:** «تست بارداری»<br>**متن:** «آزمایش خون بتا در تاریخی که کلینیک گفته»<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:187` | |
| IV-K07 | `ivf_protocols` / `antagonist` | نام و توضیح پروتکل‌ها | **عنوان:** «آنتاگونیست»<br>**متن:** «رایج‌ترین پروتکل؛ تحریک از روزهای اول سیکل و داروی جلوگیری از تخمک‌گذاری زودرس از حدود روز پنجم تحریک»<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:192` | |
| IV-K08 | `ivf_protocols` / `long_agonist` | نام و توضیح پروتکل‌ها | **عنوان:** «آگونیست طولانی»<br>**متن:** «سرکوب هورمونی از سیکل قبل، بعد تحریک»<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:197` | |
| IV-K09 | `ivf_protocols` / `short_agonist` | نام و توضیح پروتکل‌ها | **عنوان:** «آگونیست کوتاه»<br>**متن:** «آگونیست و تحریک تقریباً هم‌زمان شروع می‌شوند»<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:202` | |
| IV-K10 | `ivf_protocols` / `mild` | نام و توضیح پروتکل‌ها | **عنوان:** «تحریک ملایم»<br>**متن:** «دوز کمتر دارو و معمولاً تخمک کمتر»<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:207` | |
| IV-K11 | `ivf_protocols` / `natural` | نام و توضیح پروتکل‌ها | **عنوان:** «سیکل طبیعی»<br>**متن:** «بدون تحریک یا با داروی خیلی کم»<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:212` | |
| IV-K12 | `ivf_protocols` / `frozen_transfer` | نام و توضیح پروتکل‌ها | **عنوان:** «انتقال جنین منجمد»<br>**متن:** «آماده‌سازی رحم برای انتقال جنین فریزشده»<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:217` | |
| IV-K13 | `ivf_injection_sites` / `abdomen_upper_right` | فهرست محل‌های تزریق و توضیح هر کدام | **عنوان:** «شکم · راست بالا»<br>**متن:** «حداقل ۵ سانت دور از ناف؛ روی کبودی یا جای قبلی نزن»<br>**meta:** `{"region": "abdomen", "side": "right"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:222` | |
| IV-K14 | `ivf_injection_sites` / `abdomen_upper_left` | فهرست محل‌های تزریق و توضیح هر کدام | **عنوان:** «شکم · چپ بالا»<br>**متن:** «حداقل ۵ سانت دور از ناف؛ روی کبودی یا جای قبلی نزن»<br>**meta:** `{"region": "abdomen", "side": "left"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:227` | |
| IV-K15 | `ivf_injection_sites` / `thigh_right` | فهرست محل‌های تزریق و توضیح هر کدام | **عنوان:** «ران · راست»<br>**متن:** «جلو و بیرون ران، میانه فاصله زانو تا لگن»<br>**meta:** `{"region": "thigh", "side": "right"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:232` | |
| IV-K16 | `ivf_injection_sites` / `thigh_left` | فهرست محل‌های تزریق و توضیح هر کدام | **عنوان:** «ران · چپ»<br>**متن:** «جلو و بیرون ران، میانه فاصله زانو تا لگن»<br>**meta:** `{"region": "thigh", "side": "left"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:237` | |
| IV-K17 | `ivf_injection_sites` / `abdomen_lower_right` | فهرست محل‌های تزریق و توضیح هر کدام | **عنوان:** «شکم · راست پایین»<br>**متن:** «حداقل ۵ سانت دور از ناف؛ روی کبودی یا جای قبلی نزن»<br>**meta:** `{"region": "abdomen", "side": "right"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:242` | |
| IV-K18 | `ivf_injection_sites` / `abdomen_lower_left` | فهرست محل‌های تزریق و توضیح هر کدام | **عنوان:** «شکم · چپ پایین»<br>**متن:** «حداقل ۵ سانت دور از ناف؛ روی کبودی یا جای قبلی نزن»<br>**meta:** `{"region": "abdomen", "side": "left"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:247` | |
| IV-K19 | `ivf_injection_sites` / `arm_right` | فهرست محل‌های تزریق و توضیح هر کدام | **عنوان:** «بازو · راست»<br>**متن:** «پشت بازو؛ معمولاً وقتی کسی کمکت می‌کند»<br>**meta:** `{"region": "arm", "side": "right"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:252` | |
| IV-K20 | `ivf_injection_sites` / `arm_left` | فهرست محل‌های تزریق و توضیح هر کدام | **عنوان:** «بازو · چپ»<br>**متن:** «پشت بازو؛ معمولاً وقتی کسی کمکت می‌کند»<br>**meta:** `{"region": "arm", "side": "left"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:257` | |
| IV-K21 | `ivf_med_presets` / `fsh` | فهرست داروهای پیش‌فرض، نقش، مسیر و دوز/توضیح | **عنوان:** «هورمون تحریک (FSH)»<br>**meta:** `{"role": "stimulation", "route": "subcutaneous", "unit": "iu", "times": ["20:00"], "stock_unit": "pen"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:262` | |
| IV-K22 | `ivf_med_presets` / `hmg` | فهرست داروهای پیش‌فرض، نقش، مسیر و دوز/توضیح | **عنوان:** «hMG (FSH + LH)»<br>**meta:** `{"role": "stimulation", "route": "subcutaneous", "unit": "iu", "times": ["20:00"], "stock_unit": "vial"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:267` | |
| IV-K23 | `ivf_med_presets` / `gnrh_antagonist` | فهرست داروهای پیش‌فرض، نقش، مسیر و دوز/توضیح | **عنوان:** «داروی جلوگیری از تخمک‌گذاری زودرس (آنتاگونیست)»<br>**meta:** `{"role": "suppression", "route": "subcutaneous", "unit": "mg", "times": ["08:00"], "stock_unit": "prefilled_syringe"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:272` | |
| IV-K24 | `ivf_med_presets` / `gnrh_agonist` | فهرست داروهای پیش‌فرض، نقش، مسیر و دوز/توضیح | **عنوان:** «آگونیست GnRH»<br>**meta:** `{"role": "suppression", "route": "subcutaneous", "unit": "mg", "times": ["08:00"], "stock_unit": "vial"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:277` | |
| IV-K25 | `ivf_med_presets` / `hcg_trigger` | فهرست داروهای پیش‌فرض، نقش، مسیر و دوز/توضیح | **عنوان:** «تزریق تریگر (hCG)»<br>**meta:** `{"role": "trigger", "route": "subcutaneous", "unit": "iu", "stock_unit": "prefilled_syringe"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:282` | |
| IV-K26 | `ivf_med_presets` / `agonist_trigger` | فهرست داروهای پیش‌فرض، نقش، مسیر و دوز/توضیح | **عنوان:** «تزریق تریگر (آگونیست)»<br>**meta:** `{"role": "trigger", "route": "subcutaneous", "unit": "mg", "stock_unit": "prefilled_syringe"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:287` | |
| IV-K27 | `ivf_med_presets` / `progesterone_vaginal` | فهرست داروهای پیش‌فرض، نقش، مسیر و دوز/توضیح | **عنوان:** «پروژسترون واژینال»<br>**meta:** `{"role": "luteal_support", "route": "vaginal", "unit": "mg", "times": ["08:00", "20:00"], "stock_unit": "box"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:292` | |
| IV-K28 | `ivf_med_presets` / `progesterone_im` | فهرست داروهای پیش‌فرض، نقش، مسیر و دوز/توضیح | **عنوان:** «پروژسترون تزریقی»<br>**meta:** `{"role": "luteal_support", "route": "intramuscular", "unit": "mg", "times": ["20:00"], "stock_unit": "ampoule"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:297` | |
| IV-K29 | `ivf_med_presets` / `estradiol` | فهرست داروهای پیش‌فرض، نقش، مسیر و دوز/توضیح | **عنوان:** «استرادیول»<br>**meta:** `{"role": "luteal_support", "route": "oral", "unit": "mg", "times": ["08:00", "20:00"], "stock_unit": "box"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:302` | |
| IV-K30 | `ivf_guidance` / `trigger_timing` | متن راهنما (زمان تزریق تریگر، چرخش محل تزریق و ...) | **عنوان:** «تزریق تریگر (آخرین تزریق)»<br>**متن:** «وقتی پزشک اعلام کرد، دقیقاً در همان ساعت تزریق کن؛ زمان آن برای تخمک‌کشی مهم است.»<br>**meta:** `{"placement": "meds"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:307` | |
| IV-K31 | `ivf_guidance` / `site_rotation` | متن راهنما (زمان تزریق تریگر، چرخش محل تزریق و ...) | **عنوان:** «محل تزریق را عوض کن»<br>**متن:** «هر بار جای دیگری تزریق کن تا پوست تحریک و کبود نشود. در شکم حداقل ۵ سانت از ناف فاصله بگیر.»<br>**meta:** `{"placement": "meds"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:312` | |
| IV-K32 | `ivf_guidance` / `scan_interpretation` | متن راهنما (زمان تزریق تریگر، چرخش محل تزریق و ...) | **عنوان:** «تفسیر با پزشک است»<br>**متن:** «تفسیر و تصمیم درباره زمان تخمک‌کشی با پزشکت است. ما فقط کمک می‌کنیم همه‌چیز یک‌جا ثبت شود.»<br>**meta:** `{"placement": "scan"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:317` | |
| IV-K33 | `ivf_guidance` / `tww_feelings` | متن راهنما (زمان تزریق تریگر، چرخش محل تزریق و ...) | **عنوان:** «هر حسی داری طبیعی است»<br>**متن:** «هر حسی داری طبیعی است. تست خانگی زودتر از موعد می‌تواند گمراه‌کننده باشد.»<br>**meta:** `{"placement": "tww"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:322` | |
| IV-K34 | `ivf_guidance` / `early_test` | متن راهنما (زمان تزریق تریگر، چرخش محل تزریق و ...) | **عنوان:** «تست خانگی زودهنگام»<br>**متن:** «داروهای تریگر و پروژسترون می‌توانند جواب تست خانگی را در روزهای اول اشتباه نشان دهند؛ منتظر آزمایش خون بتا بمان.»<br>**meta:** `{"placement": "tww"}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:327` | |
| IV-K35 | `ivf_danger_signs` / `ohss` | علائم خطر (OHSS و غیره) و متن ارجاع | **عنوان:** «اگر این‌ها را داشتی فوراً به پزشک خبر بده»<br>**متن:** «نفخ شدید و سریع شکم، تنگی نفس، کم شدن ادرار، درد شدید شکم، یا خونریزی زیاد.»<br>**meta:** `{"severity": "urgent", "placement": ["home", "tww"], "hotlines": ["115"]}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:332` | |
| IV-K36 | `ivf_danger_signs` / `fever_after_procedure` | علائم خطر (OHSS و غیره) و متن ارجاع | **عنوان:** «تب بعد از تخمک‌کشی یا انتقال»<br>**متن:** «تب، لرز یا ترشح بدبو بعد از تخمک‌کشی یا انتقال جنین را همان روز به کلینیک بگو.»<br>**meta:** `{"severity": "urgent", "placement": ["home", "tww"], "hotlines": ["115"]}`<br>audiences: ttc | `backend-go/db/migrations/00027_ivf.sql:337` | |


## از دست دادن بارداری (Loss)

مسیر حمایتی پس از از دست دادن بارداری: انواع، پیگیری (بتا، ویزیت)، علائم هشدار، خطوط پشتیبانی و گام‌های بعدی. لحن در QA بازبینی شده و بازنویسی‌های پیشنهادی (جدول آخر این بخش) هنوز اعمال نشده‌اند.

### الف) قواعد، آستانه‌ها و متن‌های داخل کد

| شناسه | چه چیزی باید بررسی شود | مقدار / متن فعلی (عین کد) | منبع | تصمیم بالینی |
|---|---|---|---|---|
| LS-C01 | زمان پیشنهادی ویزیت پیگیری از تاریخ اتفاق | `VisitAfterDays = 14` («حدود ۲ هفته بعد»)؛ متن فعلی بدون زمان: «اگر پزشکت ویزیت پیگیری خواسته، اینجا ثبتش کن» | `backend-go/internal/loss/loss.go ; frontend/messages/fa/loss.json (care.visit.sub)` | |
| LS-C02 | تعداد از دست دادن‌های ثبت‌شده برای نمایش راهنمای مکرر | `RecurrentLosses = 2`؛ متن: «اگر این اولین بار نبود، می‌توانی با پزشکت درباره آزمایش‌هایی که علت را بررسی می‌کنند صحبت کنی.» | `backend-go/internal/loss/loss.go ; frontend/messages/fa/loss.json (next.recurrentFallback)` | |
| LS-C03 | ساعت یادآور آزمایش بتا، پنجرهٔ ثبت احساس، و حداکثر بازهٔ گذشتهٔ تاریخ اتفاق | `BetaHour = 9`؛ `MoodDays = 14`؛ `MaxPastDays = 365` | `backend-go/internal/loss/loss.go` | |
| LS-C04 | گزینه‌های «از اینجا به بعد» | `NextCycle`, `NextTTC`, `NextNothing` (پرسش: «چیزی نمی‌خواهم» → برگشت به چرخه) | `backend-go/internal/loss/loss.go (NextSteps)` | |
| LS-C05 | متن برگهٔ خونریزی و متن حمایت روزانه | «تا وقتی خونریزی ادامه دارد، هر روز آن را در ثبت روزانه بنویس. وقتی قطع شد، اینجا علامت بزن.» / «غم، احساس گناه یا بی‌حسی بعد از این اتفاق طبیعی است و تقصیر تو نیست.» | `frontend/messages/fa/loss.json (care.bleeding.sheetBody, care.moodSupport)` | |
| LS-C06 | محتوای بارداری پس از ثبت اتفاق: همیشه متوقف شود؟ داروها متوقف یا ویزیت‌ها لغو؟ | تصمیم باز در QUESTIONS (CB-LOSS-01) | `roadmap/PROGRESS.md (CB-LOSS-01)` | |

### ب) ردیف‌های کاتالوگ (جدول catalog_items در seed)

همهٔ ردیف‌ها با `needs_review = 1` ذخیره شده‌اند. مقادیر زیر متن فارسی seed هستند (نسخهٔ انگلیسی در همان فایل است). ادمین بعد از تأیید می‌تواند پرچم را بردارد.

| شناسه | گروه / کد | چه چیزی باید بررسی شود | مقدار / متن فعلی | منبع | تصمیم بالینی |
|---|---|---|---|---|---|
| LS-K01 | `loss_types` / `early_miscarriage` | انواع از دست دادن بارداری و توضیح هرکدام | **عنوان:** «سقط در ۳ ماه اول» | `backend-go/db/migrations/00028_loss.sql:77` | |
| LS-K02 | `loss_types` / `late_miscarriage` | انواع از دست دادن بارداری و توضیح هرکدام | **عنوان:** «سقط بعد از ۳ ماه اول» | `backend-go/db/migrations/00028_loss.sql:82` | |
| LS-K03 | `loss_types` / `ectopic` | انواع از دست دادن بارداری و توضیح هرکدام | **عنوان:** «حاملگی خارج از رحم» | `backend-go/db/migrations/00028_loss.sql:87` | |
| LS-K04 | `loss_types` / `chemical` | انواع از دست دادن بارداری و توضیح هرکدام | **عنوان:** «بارداری شیمیایی یا تست مثبت کوتاه» | `backend-go/db/migrations/00028_loss.sql:92` | |
| LS-K05 | `loss_types` / `unspecified` | انواع از دست دادن بارداری و توضیح هرکدام | **عنوان:** «ترجیح می‌دهم نگویم» | `backend-go/db/migrations/00028_loss.sql:97` | |
| LS-K06 | `loss_support` / `mood_support` | متن حمایتی و اعلان به همراه | **عنوان:** «حال دلت»<br>**متن:** «غم، احساس گناه یا بی‌حسی بعد از سقط طبیعی است و تقصیر تو نیست.» | `backend-go/db/migrations/00028_loss.sql:177` | |
| LS-K07 | `loss_support` / `crisis` | متن حمایتی و اعلان به همراه | **عنوان:** «اگر غم خیلی سنگین است»<br>**متن:** «اگر غم خیلی سنگین است یا به آسیب زدن به خودت فکر می‌کنی، همین حالا با صدای مشاور ۱۴۸۰ یا اورژانس اجتماعی ۱۲۳ تماس بگیر.»<br>**meta:** `{"severity": "urgent", "hotlines": [{"number": "1480", "label": "صدای مشاور"}, {"number": "123", "label": "اورژانس اجتماعی"}]}` | `backend-go/db/migrations/00028_loss.sql:182` | |
| LS-K08 | `loss_support` / `recurrent_hint` | متن حمایتی و اعلان به همراه | **عنوان:** «اگر سقط دوم یا سوم بود»<br>**متن:** «اگر سقط دوم یا سوم بود، درباره آزمایش‌های بررسی علت با پزشکت صحبت کن.» | `backend-go/db/migrations/00028_loss.sql:187` | |
| LS-K09 | `loss_support` / `companion_notice` | متن حمایتی و اعلان به همراه | **عنوان:** «{name} خبر داد که بارداری ادامه ندارد.»<br>**meta:** `{"someone": "همراهت"}` | `backend-go/db/migrations/00028_loss.sql:192` | |
| LS-K10 | `loss_followups` / `bleeding` | متن پیگیری (بتا، ویزیت) و زمان‌بندی | **عنوان:** «خونریزی»<br>**متن:** «ثبت روزانه تا قطع شدن» | `backend-go/db/migrations/00028_loss.sql:142` | |
| LS-K11 | `loss_followups` / `beta` | متن پیگیری (بتا، ویزیت) و زمان‌بندی | **عنوان:** «آزمایش بتا تا منفی شدن»<br>**متن:** «تکرار آزمایش را با پزشکت هماهنگ کن» | `backend-go/db/migrations/00028_loss.sql:147` | |
| LS-K12 | `loss_followups` / `visit` | متن پیگیری (بتا، ویزیت) و زمان‌بندی | **عنوان:** «ویزیت پیگیری»<br>**متن:** «حدود ۲ هفته بعد نوبت بگیر» | `backend-go/db/migrations/00028_loss.sql:152` | |
| LS-K13 | `loss_next_steps` / `cycle` | گزینه‌های «از اینجا به بعد» و گزارهٔ پزشکی هرکدام | **عنوان:** «فعلاً فقط پیگیری سیکل»<br>**متن:** «اولین پریود معمولاً ۴ تا ۶ هفته بعد می‌آید»<br>**meta:** `{"life_mode": "cycle"}` | `backend-go/db/migrations/00028_loss.sql:197` | |
| LS-K14 | `loss_next_steps` / `ttc` | گزینه‌های «از اینجا به بعد» و گزارهٔ پزشکی هرکدام | **عنوان:** «دوباره اقدام به بارداری»<br>**متن:** «زمان مناسب را با پزشکت هماهنگ کن»<br>**meta:** `{"life_mode": "ttc"}` | `backend-go/db/migrations/00028_loss.sql:202` | |
| LS-K15 | `loss_next_steps` / `nothing` | گزینه‌های «از اینجا به بعد» و گزارهٔ پزشکی هرکدام | **عنوان:** «فعلاً هیچ‌چیز»<br>**متن:** «فقط یادآور پیگیری‌های پزشکی می‌ماند»<br>**meta:** `{"life_mode": "cycle"}` | `backend-go/db/migrations/00028_loss.sql:207` | |
| LS-K16 | `loss_moods` / `sad` | گزینه‌های احساس روزانه | **عنوان:** «غمگین» | `backend-go/db/migrations/00028_loss.sql:157` | |
| LS-K17 | `loss_moods` / `numb` | گزینه‌های احساس روزانه | **عنوان:** «بی‌حس» | `backend-go/db/migrations/00028_loss.sql:162` | |
| LS-K18 | `loss_moods` / `angry` | گزینه‌های احساس روزانه | **عنوان:** «عصبانی» | `backend-go/db/migrations/00028_loss.sql:167` | |
| LS-K19 | `loss_moods` / `a_bit_better` | گزینه‌های احساس روزانه | **عنوان:** «کمی بهتر» | `backend-go/db/migrations/00028_loss.sql:172` | |
| LS-K20 | `loss_warning_signs` / `heavy_bleeding` | علائم هشدار و توصیهٔ مراجعه | **عنوان:** «خونریزی خیلی زیاد»<br>**متن:** «پر شدن ۲ نوار در ساعت، ۲ ساعت پشت سر هم»<br>**meta:** `{"severity": "urgent", "hotline": "115"}` | `backend-go/db/migrations/00028_loss.sql:102` | |
| LS-K21 | `loss_warning_signs` / `fever` | علائم هشدار و توصیهٔ مراجعه | **عنوان:** «تب ۳۸ درجه یا بیشتر»<br>**meta:** `{"severity": "urgent", "hotline": "115"}` | `backend-go/db/migrations/00028_loss.sql:107` | |
| LS-K22 | `loss_warning_signs` / `severe_pain` | علائم هشدار و توصیهٔ مراجعه | **عنوان:** «درد شدید شکم»<br>**meta:** `{"severity": "urgent", "hotline": "115"}` | `backend-go/db/migrations/00028_loss.sql:112` | |
| LS-K23 | `loss_warning_signs` / `dizziness_fainting` | علائم هشدار و توصیهٔ مراجعه | **عنوان:** «سرگیجه یا غش»<br>**meta:** `{"severity": "urgent", "hotline": "115"}` | `backend-go/db/migrations/00028_loss.sql:117` | |
| LS-K24 | `loss_warning_signs` / `foul_discharge` | علائم هشدار و توصیهٔ مراجعه | **عنوان:** «ترشح بدبو»<br>**meta:** `{"severity": "urgent", "hotline": "115"}` | `backend-go/db/migrations/00028_loss.sql:122` | |
| LS-K25 | `loss_hotlines` / `emergency` | خطوط پشتیبانی و شمارهٔ تماس | **عنوان:** «اورژانس ۱۱۵»<br>**متن:** «برای علائم خطر جسمی، همین حالا تماس بگیر»<br>**meta:** `{"number": "115", "kind": "medical"}` | `backend-go/db/migrations/00028_loss.sql:127` | |
| LS-K26 | `loss_hotlines` / `counselling` | خطوط پشتیبانی و شمارهٔ تماس | **عنوان:** «صدای مشاور ۱۴۸۰»<br>**متن:** «مشاوره تلفنی برای وقتی که حالت سنگین است»<br>**meta:** `{"number": "1480", "kind": "mental_health"}` | `backend-go/db/migrations/00028_loss.sql:132` | |
| LS-K27 | `loss_hotlines` / `social_emergency` | خطوط پشتیبانی و شمارهٔ تماس | **عنوان:** «اورژانس اجتماعی ۱۲۳»<br>**متن:** «اگر به آسیب زدن به خودت فکر می‌کنی»<br>**meta:** `{"number": "123", "kind": "crisis"}` | `backend-go/db/migrations/00028_loss.sql:137` | |

### ج) بازنویسی‌های پیشنهادی QA (اعمال‌نشده)

| شناسه | ردیف | مقدار فعلی | پیشنهاد QA | منبع | تصمیم بالینی |
|---|---|---|---|---|---|
| LS-Q01 | `loss_support/companion_notice meta.someone` | en: «Your partner» / fa: «همراهت» | «یکی از نزدیکانت» / "Someone close to you" | `docs/qa/canvas/loss.md:80-102` | |
| LS-Q02 | `loss_support/mood_support (متن)` | «…بعد از سقط…» | «…بعد از این اتفاق…» | `docs/qa/canvas/loss.md:80-102` | |
| LS-Q03 | `loss_support/recurrent_hint (عنوان و متن)` | «اگر سقط دوم یا سوم بود …» | عنوان «اگر این اولین بار نبود»؛ متن = next.recurrentFallback | `docs/qa/canvas/loss.md:80-102` | |
| LS-Q04 | `loss_followups/visit (متن)` | «حدود ۲ هفته بعد نوبت بگیر» | «اگر پزشکت ویزیت پیگیری خواسته، اینجا ثبتش کن» | `docs/qa/canvas/loss.md:80-102` | |
| LS-Q05 | `loss_next_steps/cycle (متن)` | «اولین پریود معمولاً ۴ تا ۶ هفته بعد می‌آید» | بدون تغییر؛ منتظر تأیید مرجع بالینی (گزارهٔ پزشکی) | `docs/qa/canvas/loss.md:80-102` | |


## یائسگی (Menopause)

مرحله‌بندی یائسگی، پرسشنامهٔ ماهانهٔ ۱۱ سؤالی با باندها، هشدار خونریزی بعد از یائسگی، توصیه‌ها، گروه‌های چکاپ و ۹ چکاپ مخصوص یائسگی (فاصله، سن شروع، دلیل). همهٔ آستانه‌های کد در همین بخش فهرست شده‌اند.

### الف) قواعد، آستانه‌ها و متن‌های داخل کد

| شناسه | چه چیزی باید بررسی شود | مقدار / متن فعلی (عین کد) | منبع | تصمیم بالینی |
|---|---|---|---|---|
| MP-C01 | قاعدهٔ مرحله: ≥ ۱۲ ماه بدون پریود = یائسگی؛ در پری‌منوپوز پیشنهاد «یائسگی»؛ یائسگی جراحی همیشه یائسگی | `MenopauseMonths = 12`؛ stage: peri / meno / post / unsure | `backend-go/internal/menopause/stage.go` | |
| MP-C02 | پنجرهٔ هشدار «خونریزی بعد از یائسگی»: هر خونریزی/لکه‌بینی در ۳۰ روز اخیر در مرحلهٔ meno/post | `BleedingAlertDays = 30` | `backend-go/internal/menopause/today.go` | |
| MP-C03 | آستانه‌های پیام‌ها: افزایش امتیاز، تازگی پرسشنامه، نزدیکی بازبینی HRT | `ScoreWorseBy = 4` (از ۴۴)؛ `ScoreFreshMonths = 2` ماه جلالی؛ `HRTReviewDays = 14` | `backend-go/internal/messages/menomessages/rules.go` | |
| MP-C04 | متن هشدار خونریزی (نسخهٔ داخلی) | «خونریزی بعد از یائسگی» / «خونریزی یا لکه‌بینی ثبت کرده‌ای. هر خونریزی بعد از ۱۲ ماه قطع پریود باید توسط پزشک بررسی شود. بیشتر وقت‌ها علت ساده‌ای دارد، اما بررسی زودهنگام مهم است.» / «این مورد را به پزشک بگو» | `backend-go/internal/messages/menomessages/lang/fa/menopause_messages.json` | |
| MP-C05 | متن پیام چکاپ عقب‌افتاده و چکاپ رسیده | «زمان {checkup} گذشته است. اگر زودتر نوبت بگیری، برنامه سلامتت مرتب می‌ماند.» / «زمان {checkup} رسیده است. بعد از یائسگی چکاپ منظم برای استخوان و قلب اهمیت بیشتری دارد.» | `backend-go/internal/messages/menomessages/lang/fa/menopause_messages.json (checkup_overdue, checkup_due)` | |
| MP-C06 | متن پیام افزایش امتیاز و بازبینی درمان | «امتیاز آخرت {total} است؛ {delta} واحد بیشتر از دفعه قبل ({previous}). ببین چه چیزی تغییر کرده و اگر بالا ماند با پزشکت حرف بزن.» / «{when}: بازبینی {name} با پزشکت ({date}). حال این مدت و عوارض احتمالی را یادداشت کن تا با خودت ببری.» | `backend-go/internal/messages/menomessages/lang/fa/menopause_messages.json (score_worsened, hrt_review)` | |
| MP-C07 | بازبینی پیشنهادی HRT: ۳ ماه پس از شروع (اگر تاریخ بازبینی نباشد) | `DefaultReviewAfterMonths = 3` («معمولاً ۳ ماه بعد از شروع، اثر و عوارض بررسی می‌شود») | `backend-go/internal/menopause/treatment_items.go` | |
| MP-C08 | تایمر گرگرفتگی: سقف مدت، ثبت با تأخیر، ساعت‌های «شبانه»، شدت‌ها | `MaxFlashSeconds = 3600`؛ `MaxBackfillDays = 7`؛ شبانه = ۲۲:۰۰ تا ۰۶:۰۰ تهران؛ شدت‌ها `mild, moderate, severe, very_severe` | `backend-go/internal/menopause/flashes.go` | |
| MP-C09 | محرک‌های گرگرفتگی (برچسب‌های برد) | `caffeine, stress, spicy_food, warm_room, hot_drink, exercise, unknown` («گرما» → warm_room) | `roadmap/PROGRESS.md (CB-MENO-07, CB-MENO-01)` | |
| MP-C10 | نکتهٔ تنفس هنگام گرگرفتگی (نسخهٔ جایگزین در برنامه) | «نفس عمیق و آرام (۶ بار در دقیقه) و خنک کردن مچ و گردن در لحظه کمک می‌کند.» | `frontend/messages/fa/menopause.json (hotFlash.tipFallback)` | |
| MP-C11 | الگوهای یافت‌شده در ثبت‌های خود کاربر: پنجره و حداقل داده، و دو جملهٔ الگو | پنجرهٔ ۹۰ روز (`PatternWindowDays`)؛ حداقل: `CorrMinDays = 20`، `CorrMinGroupDays = 5`، `CorrMinOutcome = 3`؛ «در روزهایی که :trigger ثبت کردی، گرگرفتگی بیشتری داشتی.» (و «کمتری») / «شب‌هایی که تعریق شبانه داشتی، روز بعد خستگی‌ات بیشتر بوده.» (و «کمتر») | `backend-go/internal/menopause/patterns.go ; backend-go/internal/analysis/stats.go ; backend-go/internal/menopause/lang/fa/menopause.json (patterns, needs_review: true)` | |
| MP-C12 | پرسشنامهٔ ماهانه: ۱۱ سؤال با پاسخ ۰ تا ۴، جمع از ۴۴ و سه حوزه؛ یک پرسشنامه در هر ماه جلالی | `DefaultItemMax = 4`؛ حوزه‌ها `somatic, psychological, urogenital` (متن سؤال‌ها و باندها در جدول کاتالوگ زیر) | `backend-go/internal/menopause/score.go` | |
| MP-C13 | هدف هفتگی استراحت/آرام‌سازی (دقیقه) و پنجرهٔ نمایش گزارش پزشک | هدف `weekly_goal: 70` دقیقه (ردیف lifestyle_relaxation در کاتالوگ)؛ گزارش پیش‌فرض ۳ ماه (`DefaultReportMonths = 3`)، ۵ علامت شایع (`TopSymptoms = 5`)؛ پرسش: ٪ علائم از روزهای ثبت‌شده یا همهٔ روزها؟ | `backend-go/internal/menopause/report.go ; roadmap/PROGRESS.md (CB-MENO-01, CB-MENO-03)` | |

### ب) ردیف‌های کاتالوگ (جدول catalog_items در seed)

همهٔ ردیف‌ها با `needs_review = 1` ذخیره شده‌اند. مقادیر زیر متن فارسی seed هستند (نسخهٔ انگلیسی در همان فایل است). ادمین بعد از تأیید می‌تواند پرچم را بردارد.

| شناسه | گروه / کد | چه چیزی باید بررسی شود | مقدار / متن فعلی | منبع | تصمیم بالینی |
|---|---|---|---|---|---|
| MP-K01 | `meno_score_items` / `hot_flashes` | متن سؤال، حوزه (جسمی/روانی/ادراری‌تناسلی) و حداکثر پاسخ | **عنوان:** «گرگرفتگی و تعریق»<br>**متن:** «موج‌های گرما و تعریق، در روز یا شب»<br>**meta:** `{"domain": "somatic", "max": 4, "log": ["symptoms.general.hot_flashes", "symptoms.general.night_sweats"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:132` | |
| MP-K02 | `meno_score_items` / `heart_discomfort` | متن سؤال، حوزه (جسمی/روانی/ادراری‌تناسلی) و حداکثر پاسخ | **عنوان:** «ناراحتی قلبی»<br>**متن:** «تپش قلب، تند یا نامنظم زدن قلب، احساس فشار در سینه»<br>**meta:** `{"domain": "somatic", "max": 4, "log": ["symptoms.general.palpitations"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:137` | |
| MP-K03 | `meno_score_items` / `sleep_problems` | متن سؤال، حوزه (جسمی/روانی/ادراری‌تناسلی) و حداکثر پاسخ | **عنوان:** «مشکلات خواب»<br>**متن:** «سخت خوابیدن، بیدار شدن شبانه یا زود بیدار شدن»<br>**meta:** `{"domain": "somatic", "max": 4, "log": ["symptoms.general.insomnia"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:142` | |
| MP-K04 | `meno_score_items` / `joint_muscle` | متن سؤال، حوزه (جسمی/روانی/ادراری‌تناسلی) و حداکثر پاسخ | **عنوان:** «درد مفاصل و عضلات»<br>**متن:** «درد یا خشکی مفاصل و عضلات»<br>**meta:** `{"domain": "somatic", "max": 4, "log": ["pain.location.joints"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:147` | |
| MP-K05 | `meno_score_items` / `depressive_mood` | متن سؤال، حوزه (جسمی/روانی/ادراری‌تناسلی) و حداکثر پاسخ | **عنوان:** «غمگینی یا بی‌حوصلگی»<br>**متن:** «احساس غم، بی‌حوصلگی، زود گریه کردن یا بی‌انگیزگی»<br>**meta:** `{"domain": "psychological", "max": 4, "log": ["symptoms.general.low_mood"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:152` | |
| MP-K06 | `meno_score_items` / `irritability` | متن سؤال، حوزه (جسمی/روانی/ادراری‌تناسلی) و حداکثر پاسخ | **عنوان:** «تحریک‌پذیری»<br>**متن:** «عصبی بودن، زود از کوره در رفتن»<br>**meta:** `{"domain": "psychological", "max": 4, "log": ["symptoms.general.irritability"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:157` | |
| MP-K07 | `meno_score_items` / `anxiety` | متن سؤال، حوزه (جسمی/روانی/ادراری‌تناسلی) و حداکثر پاسخ | **عنوان:** «اضطراب»<br>**متن:** «بی‌قراری، دلشوره یا احساس ترس»<br>**meta:** `{"domain": "psychological", "max": 4, "log": ["symptoms.general.anxiety"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:162` | |
| MP-K08 | `meno_score_items` / `exhaustion` | متن سؤال، حوزه (جسمی/روانی/ادراری‌تناسلی) و حداکثر پاسخ | **عنوان:** «خستگی جسمی و ذهنی»<br>**متن:** «کم شدن انرژی، تمرکز و حافظه»<br>**meta:** `{"domain": "psychological", "max": 4, "log": ["symptoms.general.fatigue", "symptoms.general.brain_fog"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:167` | |
| MP-K09 | `meno_score_items` / `sexual_problems` | متن سؤال، حوزه (جسمی/روانی/ادراری‌تناسلی) و حداکثر پاسخ | **عنوان:** «مشکلات جنسی»<br>**متن:** «تغییر در میل، رابطه یا رضایت جنسی»<br>**meta:** `{"domain": "urogenital", "max": 4, "log": ["urogenital.symptoms.low_libido"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:172` | |
| MP-K10 | `meno_score_items` / `bladder_problems` | متن سؤال، حوزه (جسمی/روانی/ادراری‌تناسلی) و حداکثر پاسخ | **عنوان:** «مشکلات ادراری»<br>**متن:** «تکرر، فوریت یا نشت ادرار»<br>**meta:** `{"domain": "urogenital", "max": 4, "log": ["urogenital.symptoms.bladder_symptoms"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:177` | |
| MP-K11 | `meno_score_items` / `vaginal_dryness` | متن سؤال، حوزه (جسمی/روانی/ادراری‌تناسلی) و حداکثر پاسخ | **عنوان:** «خشکی واژن»<br>**متن:** «احساس خشکی یا سوزش، یا درد هنگام رابطه»<br>**meta:** `{"domain": "urogenital", "max": 4, "log": ["urogenital.symptoms.vaginal_dryness"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:182` | |
| MP-K12 | `meno_score_bands` / `none` | آستانه‌های کل امتیاز (از ۴۴) و برچسب هر باند | **عنوان:** «بدون علامت»<br>**meta:** `{"min": 0, "max": 4}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:187` | |
| MP-K13 | `meno_score_bands` / `mild` | آستانه‌های کل امتیاز (از ۴۴) و برچسب هر باند | **عنوان:** «خفیف»<br>**meta:** `{"min": 5, "max": 8}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:192` | |
| MP-K14 | `meno_score_bands` / `moderate` | آستانه‌های کل امتیاز (از ۴۴) و برچسب هر باند | **عنوان:** «متوسط»<br>**meta:** `{"min": 9, "max": 16}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:197` | |
| MP-K15 | `meno_score_bands` / `severe` | آستانه‌های کل امتیاز (از ۴۴) و برچسب هر باند | **عنوان:** «شدید»<br>**meta:** `{"min": 17, "max": 44}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:202` | |
| MP-K16 | `meno_alerts` / `postmenopausal_bleeding` | متن هشدار و شرط ارجاع به پزشک | **عنوان:** «خونریزی بعد از یائسگی»<br>**متن:** «هر خونریزی یا لکه‌بینی، حتی کم، بعد از ۱۲ ماه قطع پریود باید بررسی شود. بیشتر وقت‌ها علت ساده‌ای دارد، اما بررسی زودهنگام مهم است.»<br>**meta:** `{"severity": "urgent", "primary": true, "stages": ["meno", "post"], "cta": "این مورد را به پزشک بگو"}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:207` | |
| MP-K17 | `meno_alerts` / `heavy_perimenopause_bleeding` | متن هشدار و شرط ارجاع به پزشک | **عنوان:** «خونریزی خیلی زیاد یا طولانی در پیش‌یائسگی»<br>**متن:** «بیش از ۷ روز، یا پر شدن نوار در کمتر از ۲ ساعت»<br>**meta:** `{"severity": "caution", "stages": ["peri", "unsure"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:212` | |
| MP-K18 | `meno_alerts` / `chest_pain_palpitations` | متن هشدار و شرط ارجاع به پزشک | **عنوان:** «درد قفسه سینه یا تپش قلب شدید»<br>**متن:** «اگر ناگهانی و شدید است، با اورژانس (۱۱۵) تماس بگیر»<br>**meta:** `{"severity": "urgent", "emergency": true, "hotline": "115"}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:217` | |
| MP-K19 | `meno_alerts` / `one_sided_leg_swelling` | متن هشدار و شرط ارجاع به پزشک | **عنوان:** «درد و ورم یک‌طرفه ساق پا»<br>**متن:** «به‌خصوص اگر هورمون‌درمانی می‌کنی»<br>**meta:** `{"severity": "urgent", "hrt": true}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:222` | |
| MP-K20 | `meno_alerts` / `breast_change` | متن هشدار و شرط ارجاع به پزشک | **عنوان:** «توده یا تغییر در پستان»<br>**متن:** «هر تغییر تازه را به پزشک نشان بده»<br>**meta:** `{"severity": "caution"}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:227` | |
| MP-K21 | `meno_alerts` / `persistent_low_mood` | متن هشدار و شرط ارجاع به پزشک | **عنوان:** «غم عمیق یا ناامیدی چند هفته‌ای»<br>**متن:** «حرف زدن با پزشک یا مشاور کمک می‌کند»<br>**meta:** `{"severity": "caution"}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:232` | |
| MP-K22 | `meno_alerts` / `fragility_fracture` | متن هشدار و شرط ارجاع به پزشک | **عنوان:** «شکستگی با ضربه یا زمین خوردن ساده»<br>**متن:** «ممکن است نشانه پوکی استخوان باشد»<br>**meta:** `{"severity": "caution"}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:237` | |
| MP-K23 | `meno_tips` / `stage_peri` | متن توصیه، مرحلهٔ مشمول و تنظیمات (مثلاً هدف هفتگی) | **عنوان:** «پیش‌یائسگی»<br>**متن:** «پریودها ممکن است نامنظم، کم یا زیاد شوند و علائم تازه بیایند. ثبت پریود و علائم کمک می‌کند الگوها را ببینی و با پزشک دقیق‌تر حرف بزنی.»<br>**meta:** `{"placement": "home", "stages": ["peri"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:242` | |
| MP-K24 | `meno_tips` / `stage_meno` | متن توصیه، مرحلهٔ مشمول و تنظیمات (مثلاً هدف هفتگی) | **عنوان:** «یائسگی»<br>**متن:** «از ۱۲ ماه گذشته، یعنی وارد یائسگی شده‌ای. از این به بعد هر خونریزی یا لکه‌بینی را ثبت کن و به پزشک خبر بده.»<br>**meta:** `{"placement": "home", "stages": ["meno"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:247` | |
| MP-K25 | `meno_tips` / `stage_post` | متن توصیه، مرحلهٔ مشمول و تنظیمات (مثلاً هدف هفتگی) | **عنوان:** «پس از یائسگی»<br>**متن:** «در این سال‌ها مراقبت از استخوان و قلب مهم‌تر می‌شود. چکاپ‌های منظم را دنبال کن و هر خونریزی یا لکه‌بینی را به پزشک بگو.»<br>**meta:** `{"placement": "home", "stages": ["post"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:252` | |
| MP-K26 | `meno_tips` / `stage_unsure` | متن توصیه، مرحلهٔ مشمول و تنظیمات (مثلاً هدف هفتگی) | **عنوان:** «مطمئن نیستم»<br>**متن:** «اگر مطمئن نیستی کجای مسیر هستی، تاریخ تقریبی آخرین پریود کمک می‌کند. ۱۲ ماه بدون پریود یعنی یائسگی؛ پزشک می‌تواند دقیق‌تر بگوید.»<br>**meta:** `{"placement": "home", "stages": ["unsure"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:257` | |
| MP-K27 | `meno_tips` / `log_bleeding_note` | متن توصیه، مرحلهٔ مشمول و تنظیمات (مثلاً هدف هفتگی) | **عنوان:** «خونریزی بعد از یائسگی»<br>**متن:** «بعد از یائسگی هر خونریزی باید به پزشک گفته شود؛ اگر ثبت کنی راهنمایی‌ات می‌کنیم.»<br>**meta:** `{"placement": "log", "stages": ["meno", "post"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:262` | |
| MP-K28 | `meno_tips` / `hot_flash_breathing` | متن توصیه، مرحلهٔ مشمول و تنظیمات (مثلاً هدف هفتگی) | **عنوان:** «در لحظه گرگرفتگی»<br>**متن:** «نفس عمیق و آرام (۶ بار در دقیقه) و خنک کردن مچ و گردن در لحظه کمک می‌کند.»<br>**meta:** `{"placement": "hot_flash"}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:267` | |
| MP-K29 | `meno_tips` / `hrt_review` | متن توصیه، مرحلهٔ مشمول و تنظیمات (مثلاً هدف هفتگی) | **عنوان:** «بازبینی با پزشک»<br>**متن:** «معمولاً ۳ ماه بعد از شروع، اثر و عوارض بررسی می‌شود.»<br>**meta:** `{"placement": "treatment", "review_after_months": 3}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:272` | |
| MP-K30 | `meno_tips` / `hrt_spotting` | متن توصیه، مرحلهٔ مشمول و تنظیمات (مثلاً هدف هفتگی) | **عنوان:** «لکه‌بینی در ماه‌های اول»<br>**متن:** «لکه‌بینی در ماه‌های اول شایع است، اما اگر ادامه داشت یا زیاد بود به پزشک بگو.»<br>**meta:** `{"placement": "treatment"}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:277` | |
| MP-K31 | `meno_tips` / `doctor_only` | متن توصیه، مرحلهٔ مشمول و تنظیمات (مثلاً هدف هفتگی) | **عنوان:** «فقط با نظر پزشک»<br>**متن:** «شروع، قطع یا تغییر دوز هر دارو فقط با نظر پزشک.»<br>**meta:** `{"placement": "treatment"}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:282` | |
| MP-K32 | `meno_tips` / `lifestyle_resistance` | متن توصیه، مرحلهٔ مشمول و تنظیمات (مثلاً هدف هفتگی) | **عنوان:** «ورزش مقاومتی»<br>**متن:** «برای استخوان و عضله»<br>**meta:** `{"placement": "treatment_lifestyle", "weekly_goal": 2, "goal_unit": "sessions"}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:287` | |
| MP-K33 | `meno_tips` / `lifestyle_brisk_walk` | متن توصیه، مرحلهٔ مشمول و تنظیمات (مثلاً هدف هفتگی) | **عنوان:** «پیاده‌روی تند»<br>**متن:** «۱۵۰ دقیقه در هفته»<br>**meta:** `{"placement": "treatment_lifestyle", "weekly_goal": 150, "goal_unit": "minutes"}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:292` | |
| MP-K34 | `meno_tips` / `lifestyle_relaxation` | متن توصیه، مرحلهٔ مشمول و تنظیمات (مثلاً هدف هفتگی) | **عنوان:** «تمرین آرام‌سازی»<br>**متن:** «۱۰ دقیقه، برای گرگرفتگی و خواب»<br>**meta:** `{"placement": "treatment_lifestyle", "weekly_goal": 70, "goal_unit": "minutes"}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:297` | |
| MP-K35 | `meno_tips` / `checkups_intro` | متن توصیه، مرحلهٔ مشمول و تنظیمات (مثلاً هدف هفتگی) | **عنوان:** «چکاپ‌ها و آزمایش‌ها»<br>**متن:** «بعد از یائسگی خطر پوکی استخوان و بیماری قلبی بیشتر می‌شود. این فهرست یادآور است؛ زمان‌بندی دقیق را پزشکت تعیین می‌کند.»<br>**meta:** `{"placement": "checkups"}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:302` | |
| MP-K36 | `meno_tips` / `patterns_disclaimer` | متن توصیه، مرحلهٔ مشمول و تنظیمات (مثلاً هدف هفتگی) | **عنوان:** «الگوهایی که دیدیم»<br>**متن:** «بر اساس ثبت‌های خودت · تشخیص پزشکی نیست»<br>**meta:** `{"placement": "score"}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:307` | |
| MP-K37 | `meno_checkup_groups` / `heart_metabolic` | گروه‌بندی چکاپ‌ها (کدام چکاپ در کدام گروه) | **عنوان:** «قلب و متابولیسم»<br>**meta:** `{"checkups": ["meno_blood_pressure", "meno_blood_sugar", "meno_lipids", "meno_weight_waist"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:312` | |
| MP-K38 | `meno_checkup_groups` / `bone` | گروه‌بندی چکاپ‌ها (کدام چکاپ در کدام گروه) | **عنوان:** «استخوان»<br>**meta:** `{"checkups": ["meno_bone_density", "meno_vitamin_d_calcium"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:317` | |
| MP-K39 | `meno_checkup_groups` / `cancer_screening` | گروه‌بندی چکاپ‌ها (کدام چکاپ در کدام گروه) | **عنوان:** «غربالگری سرطان»<br>**meta:** `{"checkups": ["mammography", "pap_smear", "meno_colon_screening"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:322` | |
| MP-K40 | `meno_checkup_groups` / `other` | گروه‌بندی چکاپ‌ها (کدام چکاپ در کدام گروه) | **عنوان:** «سایر»<br>**meta:** `{"checkups": ["meno_thyroid", "meno_eye_exam", "dentist"]}`<br>audiences: menopause | `backend-go/db/migrations/00022_menopause.sql:327` | |
| MP-K41 | `checkup_types` / `meno_blood_pressure` | فاصلهٔ چکاپ، سن شروع، دلیل و آماده‌سازی | **عنوان:** «فشار خون»<br>**زیرعنوان:** «ماهانه در خانه، سالانه نزد پزشک»<br>**چرایی:** «بعد از یائسگی خطر فشار خون بالا بیشتر می‌شود و اغلب بی‌علامت است. اندازه‌گیری منظم در خانه تغییرها را زود نشان می‌دهد.»<br>**دسته:** monthly؛ **انجام‌دهنده:** self؛ **فاصله (ماه):** 1؛ **حداقل سن:** NULL؛ **یادآور (روز قبل):** 3<br>**آماده‌سازی:** «۵ دقیقه آرام بنشین و بعد اندازه بگیر» | `backend-go/db/migrations/00022_menopause.sql:334` (فعال‌سازی: `00024_activate_menopause_checkups.sql`) | |
| MP-K42 | `checkup_types` / `meno_blood_sugar` | فاصلهٔ چکاپ، سن شروع، دلیل و آماده‌سازی | **عنوان:** «قند خون ناشتا یا HbA1c»<br>**زیرعنوان:** «معمولاً هر ۱ تا ۳ سال»<br>**چرایی:** «با بالا رفتن سن و تغییرات هورمونی، خطر دیابت بیشتر می‌شود. آزمایش منظم آن را زود نشان می‌دهد.»<br>**دسته:** multi_year؛ **انجام‌دهنده:** lab؛ **فاصله (ماه):** 12 تا 36؛ **حداقل سن:** NULL؛ **یادآور (روز قبل):** 14<br>**آماده‌سازی:** «برای قند ناشتا ۸ تا ۱۰ ساعت چیزی جز آب نخور» | `backend-go/db/migrations/00022_menopause.sql:339` (فعال‌سازی: `00024_activate_menopause_checkups.sql`) | |
| MP-K43 | `checkup_types` / `meno_lipids` | فاصلهٔ چکاپ، سن شروع، دلیل و آماده‌سازی | **عنوان:** «چربی خون (کلسترول و تری‌گلیسرید)»<br>**زیرعنوان:** «معمولاً هر ۱ تا ۵ سال»<br>**چرایی:** «بعد از یائسگی کلسترول معمولاً بالا می‌رود و خطر بیماری قلبی بیشتر می‌شود.»<br>**دسته:** multi_year؛ **انجام‌دهنده:** lab؛ **فاصله (ماه):** 12 تا 60؛ **حداقل سن:** NULL؛ **یادآور (روز قبل):** 14<br>**آماده‌سازی:** «اگر پزشک گفته، ۹ تا ۱۲ ساعت ناشتا باش» | `backend-go/db/migrations/00022_menopause.sql:344` (فعال‌سازی: `00024_activate_menopause_checkups.sql`) | |
| MP-K44 | `checkup_types` / `meno_weight_waist` | فاصلهٔ چکاپ، سن شروع، دلیل و آماده‌سازی | **عنوان:** «وزن و دور کمر»<br>**زیرعنوان:** «ماهانه»<br>**چرایی:** «در یائسگی چربی بیشتر دور شکم جمع می‌شود که با خطر قلبی همراه است. اندازه‌گیری ماهانه تغییرها را نشان می‌دهد.»<br>**دسته:** monthly؛ **انجام‌دهنده:** self؛ **فاصله (ماه):** 1؛ **حداقل سن:** NULL؛ **یادآور (روز قبل):** 3 | `backend-go/db/migrations/00022_menopause.sql:349` (فعال‌سازی: `00024_activate_menopause_checkups.sql`) | |
| MP-K45 | `checkup_types` / `meno_bone_density` | فاصلهٔ چکاپ، سن شروع، دلیل و آماده‌سازی | **عنوان:** «سنجش تراکم استخوان (DEXA)»<br>**زیرعنوان:** «معمولاً از ۶۵ سالگی، یا زودتر با عامل خطر»<br>**چرایی:** «بعد از یائسگی استخوان‌ها سریع‌تر تحلیل می‌روند. این سنجش پوکی استخوان را پیش از شکستگی نشان می‌دهد. اگر عامل خطر داری، زودتر با پزشک هماهنگ کن.»<br>**دسته:** age_based؛ **انجام‌دهنده:** lab؛ **فاصله (ماه):** 24؛ **حداقل سن:** 65؛ **یادآور (روز قبل):** 30 | `backend-go/db/migrations/00022_menopause.sql:354` (فعال‌سازی: `00024_activate_menopause_checkups.sql`) | |
| MP-K46 | `checkup_types` / `meno_vitamin_d_calcium` | فاصلهٔ چکاپ، سن شروع، دلیل و آماده‌سازی | **عنوان:** «ویتامین D و کلسیم»<br>**زیرعنوان:** «طبق نظر پزشک»<br>**چرایی:** «ویتامین D و کلسیم کافی برای استخوان مهم است؛ پزشک تصمیم می‌گیرد آزمایش یا مکمل لازم است یا نه.»<br>**دسته:** annual؛ **انجام‌دهنده:** lab؛ **فاصله (ماه):** 12؛ **حداقل سن:** NULL؛ **یادآور (روز قبل):** 14 | `backend-go/db/migrations/00022_menopause.sql:359` (فعال‌سازی: `00024_activate_menopause_checkups.sql`) | |
| MP-K47 | `checkup_types` / `meno_colon_screening` | فاصلهٔ چکاپ، سن شروع، دلیل و آماده‌سازی | **عنوان:** «غربالگری سرطان روده»<br>**زیرعنوان:** «از ۴۵ سالگی، طبق روش انتخابی»<br>**چرایی:** «غربالگری روده می‌تواند پولیپ یا سرطان را زود پیدا کند. فاصله تکرار به روش (آزمایش مدفوع یا کولونوسکوپی) بستگی دارد.»<br>**دسته:** age_based؛ **انجام‌دهنده:** doctor؛ **فاصله (ماه):** 12 تا 120؛ **حداقل سن:** 45؛ **یادآور (روز قبل):** 30 | `backend-go/db/migrations/00022_menopause.sql:364` (فعال‌سازی: `00024_activate_menopause_checkups.sql`) | |
| MP-K48 | `checkup_types` / `meno_thyroid` | فاصلهٔ چکاپ، سن شروع، دلیل و آماده‌سازی | **عنوان:** «تیروئید (TSH)»<br>**زیرعنوان:** «اگر خستگی یا تغییر وزن داری»<br>**چرایی:** «مشکلات تیروئید در این سن شایع‌اند و علائمشان می‌تواند شبیه یائسگی باشد.»<br>**دسته:** annual؛ **انجام‌دهنده:** lab؛ **فاصله (ماه):** 12؛ **حداقل سن:** NULL؛ **یادآور (روز قبل):** 14 | `backend-go/db/migrations/00022_menopause.sql:369` (فعال‌سازی: `00024_activate_menopause_checkups.sql`) | |
| MP-K49 | `checkup_types` / `meno_eye_exam` | فاصلهٔ چکاپ، سن شروع، دلیل و آماده‌سازی | **عنوان:** «معاینه چشم»<br>**زیرعنوان:** «سالانه»<br>**چرایی:** «معاینه منظم چشم تغییرات بینایی و بیماری‌هایی مثل آب سیاه را زود نشان می‌دهد.»<br>**دسته:** annual؛ **انجام‌دهنده:** doctor؛ **فاصله (ماه):** 12؛ **حداقل سن:** NULL؛ **یادآور (روز قبل):** 30 | `backend-go/db/migrations/00022_menopause.sql:374` (فعال‌سازی: `00024_activate_menopause_checkups.sql`) | |


## بیماری‌ها و برنامه‌ها: درد/اندومتریوز، PMDD، خونریزی سنگین PBAC (Conditions)

برنامهٔ درد/اندومتریوز، پرسشنامهٔ PMDD با نمودار لوتئال/فولیکولار، جدول PBAC برای خونریزی سنگین و نادج‌هایی که به برنامه یا پزشک ارجاع می‌دهند.

### الف) قواعد، آستانه‌ها و متن‌های داخل کد

| شناسه | چه چیزی باید بررسی شود | مقدار / متن فعلی (عین کد) | منبع | تصمیم بالینی |
|---|---|---|---|---|
| CN-C01 | جدول PBAC (Higham): امتیاز هر سطح خیس‌شدگی نوار و لخته، و آستانهٔ هشدار | نوار: کمی خیس ۱، نیمه‌خیس ۵، کاملاً خیس ۲۰؛ لخته: کوچک ۱، بزرگ ۵؛ نشت به لباس ۵؛ `AlertScore = 100` (پیام pbac_over_100)؛ `PeriodSpan = 10` روز؛ `MaxPads = 50` (برد: «لخته کوچک» با جمع ۷۱ ناسازگار است؛ seed بدون لخته) | `backend-go/internal/conditions/pbac.go ; roadmap/PROGRESS.md (CB-COND-05)` | |
| CN-C02 | نگاشت نمرهٔ درد ۱ تا ۱۰ به سطح خفیف/متوسط/شدید | ۱–۳ خفیف (`mild`)، ۴–۶ متوسط (`moderate`)، ۷–۱۰ شدید (`severe`)؛ سطح در ستون‌های درد قدیمی daily_health_logs نوشته می‌شود | `backend-go/internal/conditions/pain.go (LevelFor)` | |
| CN-C03 | پرسشنامهٔ PMDD: مقیاس، تعداد چرخهٔ لازم، پنجره‌های لوتئال/فولیکولار و آستانهٔ «الگو» | مقیاس ۱ تا ۶؛ `RequiredCycles = 2`؛ `LateLutealDays = 10` (۱۰ روز آخر چرخه)؛ فولیکولار روز ۴ تا ۱۰؛ چرخهٔ کامل: ≥ ۷ روز لوتئال و ≥ ۴ روز فولیکولار امتیازدهی‌شده؛ قابل‌ارزیابی: `EvalDays = 3` در هر پنجره؛ `LutealRise = 1.3` (میانگین لوتئال ≥ ۱٫۳ برابر فولیکولار)؛ میانگین روزانه فقط روی آیتم‌های امتیازداده‌شده | `backend-go/internal/conditions/pmdd.go` | |
| CN-C04 | نادجِ درد شدید: درد ≥ ۷ در ≥ ۲ روز چرخهٔ جاری (برای کسی که در برنامهٔ اندومتریوز نیست) | `PainMinScore = 7`، `MinDays = 2`، `MaxCycleDays = 45`، `FallbackCycleDays = 28`؛ حالت‌های مشمول: cycle, ttc, teen | `backend-go/internal/messages/conditionnudges/rules.go` | |
| CN-C05 | نادجِ خونریزی سنگین: فلوی `heavy` یا `very_heavy` در ≥ ۲ روز چرخه (پرسش: آیا PBAC ≥ ۱۰۰ هم باید نادج بدهد؟ نمایش در حالت نوجوان؟) | `heavyFlow = [heavy, very_heavy]`، `MinDays = 2` | `backend-go/internal/messages/conditionnudges/rules.go ; roadmap/PROGRESS.md (CB-COND-06)` | |
| CN-C06 | متن دو نادج | heavy_pain: «درد شدید در چند روز» / «در این چرخه {days} روز درد ۷ یا بیشتر ثبت کرده‌ای. دفترچه درد کمک می‌کند تو و پزشکت الگوی آن را ببینید.»؛ heavy_bleeding: «خونریزی سنگین در چند روز» / «در این چرخه {days} روز خونریزی سنگین ثبت کرده‌ای. جدول نوار بهداشتی (PBAC) مقدار خونریزی را می‌سنجد تا بشود بررسی‌اش کرد.» | `backend-go/internal/messages/conditionnudges/lang/fa/condition_nudges.json` | |

### ب) ردیف‌های کاتالوگ (جدول catalog_items در seed)

همهٔ ردیف‌ها با `needs_review = 1` ذخیره شده‌اند. مقادیر زیر متن فارسی seed هستند (نسخهٔ انگلیسی در همان فایل است). ادمین بعد از تأیید می‌تواند پرچم را بردارد.

| شناسه | گروه / کد | چه چیزی باید بررسی شود | مقدار / متن فعلی | منبع | تصمیم بالینی |
|---|---|---|---|---|---|
| CN-K01 | `condition_programs` / `endo` | متن معرفی برنامه و شرط ورود | **عنوان:** «اندومتریوز»<br>**متن:** «دفترچه درد با محل و شدت»<br>**meta:** `{"logs": "محل درد، شدت، نوع درد، اثر دارو"}` | `backend-go/db/migrations/00023_condition_programs.sql:89` | |
| CN-K02 | `condition_programs` / `pmdd` | متن معرفی برنامه و شرط ورود | **عنوان:** «اختلال شدید پیش از قاعدگی (PMDD)»<br>**متن:** «پرسشنامه روزانه خلق برای ۲ سیکل»<br>**meta:** `{"logs": "خلق، اضطراب، تحریک‌پذیری، تمرکز"}` | `backend-go/db/migrations/00023_condition_programs.sql:94` | |
| CN-K03 | `condition_programs` / `heavy_bleeding` | متن معرفی برنامه و شرط ورود | **عنوان:** «خونریزی شدید»<br>**متن:** «جدول امتیاز خونریزی و خطر کم‌خونی»<br>**meta:** `{"logs": "تعداد و میزان خیس شدن نوار، لخته"}` | `backend-go/db/migrations/00023_condition_programs.sql:99` | |
| CN-K04 | `condition_programs` / `pcos` | متن معرفی برنامه و شرط ورود | **عنوان:** «سندرم تخمدان پلی‌کیستیک»<br>**متن:** «نظم سیکل، پوست و مو، وزن و قند»<br>**meta:** `{"logs": "فاصله پریودها، آکنه، موهای زائد، وزن"}` | `backend-go/db/migrations/00023_condition_programs.sql:104` | |
| CN-K05 | `pain_types` / `cramping` | فهرست انواع درد | **عنوان:** «گرفتگی» | `backend-go/db/migrations/00023_condition_programs.sql:109` | |
| CN-K06 | `pain_types` / `stabbing` | فهرست انواع درد | **عنوان:** «تیر کشنده» | `backend-go/db/migrations/00023_condition_programs.sql:114` | |
| CN-K07 | `pain_types` / `burning` | فهرست انواع درد | **عنوان:** «سوزشی» | `backend-go/db/migrations/00023_condition_programs.sql:119` | |
| CN-K08 | `pain_types` / `dull_heavy` | فهرست انواع درد | **عنوان:** «مبهم و سنگین» | `backend-go/db/migrations/00023_condition_programs.sql:124` | |
| CN-K09 | `pain_associated` / `dyspareunia` | فهرست علائم همراه درد | **عنوان:** «درد در رابطه»<br>**meta:** `{"log": "sex.symptoms.pain_during_intercourse"}` | `backend-go/db/migrations/00023_condition_programs.sql:129` | |
| CN-K10 | `pain_associated` / `dyschezia` | فهرست علائم همراه درد | **عنوان:** «درد هنگام اجابت مزاج» | `backend-go/db/migrations/00023_condition_programs.sql:134` | |
| CN-K11 | `pain_associated` / `dysuria` | فهرست علائم همراه درد | **عنوان:** «درد هنگام ادرار»<br>**meta:** `{"log": "urogenital.symptoms.urination_burning"}` | `backend-go/db/migrations/00023_condition_programs.sql:139` | |
| CN-K12 | `pain_associated` / `bloating` | فهرست علائم همراه درد | **عنوان:** «نفخ»<br>**meta:** `{"log": "symptoms.digestive.bloating"}` | `backend-go/db/migrations/00023_condition_programs.sql:144` | |
| CN-K13 | `pain_associated` / `nausea` | فهرست علائم همراه درد | **عنوان:** «تهوع»<br>**meta:** `{"log": "symptoms.digestive.nausea"}` | `backend-go/db/migrations/00023_condition_programs.sql:149` | |
| CN-K14 | `pmdd_items` / `sadness` | آیتم‌های پرسشنامهٔ روزانهٔ PMDD (متن و حوزه) | **عنوان:** «غمگینی یا ناامیدی» | `backend-go/db/migrations/00023_condition_programs.sql:154` | |
| CN-K15 | `pmdd_items` / `anxiety` | آیتم‌های پرسشنامهٔ روزانهٔ PMDD (متن و حوزه) | **عنوان:** «اضطراب یا تنش» | `backend-go/db/migrations/00023_condition_programs.sql:159` | |
| CN-K16 | `pmdd_items` / `mood_swings` | آیتم‌های پرسشنامهٔ روزانهٔ PMDD (متن و حوزه) | **عنوان:** «نوسان خلق و زودرنجی» | `backend-go/db/migrations/00023_condition_programs.sql:164` | |
| CN-K17 | `pmdd_items` / `anger` | آیتم‌های پرسشنامهٔ روزانهٔ PMDD (متن و حوزه) | **عنوان:** «عصبانیت یا درگیری با دیگران» | `backend-go/db/migrations/00023_condition_programs.sql:169` | |
| CN-K18 | `pmdd_items` / `loss_of_interest` | آیتم‌های پرسشنامهٔ روزانهٔ PMDD (متن و حوزه) | **عنوان:** «بی‌علاقگی به کارهای معمول» | `backend-go/db/migrations/00023_condition_programs.sql:174` | |
| CN-K19 | `pmdd_items` / `concentration` | آیتم‌های پرسشنامهٔ روزانهٔ PMDD (متن و حوزه) | **عنوان:** «سختی در تمرکز» | `backend-go/db/migrations/00023_condition_programs.sql:179` | |
| CN-K20 | `condition_alerts` / `not_a_diagnosis` | متن هشدار، آستانه یا جملهٔ الگو | **عنوان:** «این برنامه‌ها تشخیص نمی‌دهند»<br>**متن:** «کمک می‌کنند با داده دقیق نزد پزشک بروی.»<br>**meta:** `{"severity": "info"}` | `backend-go/db/migrations/00023_condition_programs.sql:184` | |
| CN-K21 | `condition_alerts` / `pain_scale` | متن هشدار، آستانه یا جملهٔ الگو | **عنوان:** «شدت درد از ۰ تا ۱۰»<br>**متن:** «۰ یعنی بدون درد و ۱۰ بدترین درد ممکن. درد شدید یا ناگهانی، تب یا غش را همان روز به پزشک بگو.»<br>**meta:** `{"severity": "info"}` | `backend-go/db/migrations/00023_condition_programs.sql:189` | |
| CN-K22 | `condition_alerts` / `pbac_scoring` | متن هشدار، آستانه یا جملهٔ الگو | **عنوان:** «امتیاز خونریزی چطور حساب می‌شود؟»<br>**متن:** «هر نوار کمی خیس ۱ امتیاز، نیمه خیس ۵ و کاملاً خیس ۲۰ امتیاز دارد؛ لخته کوچک ۱، لخته بزرگ ۵ و نشت از نوار به لباس ۵ امتیاز. جمع یک پریود بالای ۱۰۰ معمولاً یعنی خونریزی شدید.»<br>**meta:** `{"severity": "info"}` | `backend-go/db/migrations/00023_condition_programs.sql:194` | |
| CN-K23 | `condition_alerts` / `pbac_over_100` | متن هشدار، آستانه یا جملهٔ الگو | **عنوان:** «امتیاز این پریود از ۱۰۰ بیشتر شده»<br>**متن:** «معمولاً یعنی خونریزی شدید است. به پزشک بگو و درباره آزمایش کم‌خونی (هموگلوبین و فریتین) بپرس.»<br>**meta:** `{"severity": "caution"}` | `backend-go/db/migrations/00023_condition_programs.sql:199` | |
| CN-K24 | `condition_alerts` / `pmdd_needs_two_cycles` | متن هشدار، آستانه یا جملهٔ الگو | **عنوان:** «برای جمع‌بندی، ۲ سیکل ثبت کامل لازم است»<br>**متن:** «یک سیکل کامل یعنی دست‌کم ۷ روز از ۱۰ روز آخر سیکل و ۴ روز از روزهای ۴ تا ۱۰ سیکل را ثبت کرده باشی.»<br>**meta:** `{"severity": "info"}` | `backend-go/db/migrations/00023_condition_programs.sql:204` | |
| CN-K25 | `condition_alerts` / `pmdd_pattern_luteal` | متن هشدار، آستانه یا جملهٔ الگو | **عنوان:** «الگوی تو»<br>**متن:** «علائم در ۱۰ روز آخر سیکل بالا می‌رود و با شروع پریود کم می‌شود.»<br>**meta:** `{"severity": "info"}` | `backend-go/db/migrations/00023_condition_programs.sql:209` | |
| CN-K26 | `condition_alerts` / `pmdd_pattern_unclear` | متن هشدار، آستانه یا جملهٔ الگو | **عنوان:** «الگوی تو»<br>**متن:** «هنوز الگوی روشنی بین ۱۰ روز آخر سیکل و هفته بعد از پریود دیده نمی‌شود. ثبت روزانه را ادامه بده.»<br>**meta:** `{"severity": "info"}` | `backend-go/db/migrations/00023_condition_programs.sql:214` | |
| CN-K27 | `condition_alerts` / `pmdd_not_enough_data` | متن هشدار، آستانه یا جملهٔ الگو | **عنوان:** «هنوز داده کافی نیست»<br>**متن:** «برای دیدن الگو، هم در هفته بعد از پریود و هم در ۱۰ روز آخر سیکل هر روز ثبت کن.»<br>**meta:** `{"severity": "info"}` | `backend-go/db/migrations/00023_condition_programs.sql:219` | |
| CN-K28 | `condition_alerts` / `pmdd_crisis` | متن هشدار، آستانه یا جملهٔ الگو | **عنوان:** «اگر به آسیب زدن به خودت فکر می‌کنی»<br>**متن:** «همین حالا با صدای مشاور ۱۴۸۰ یا اورژانس اجتماعی ۱۲۳ تماس بگیر.»<br>**meta:** `{"severity": "urgent", "hotlines": [{"number": "1480", "label": "صدای مشاور"}, {"number": "123", "label": "اورژانس اجتماعی"}]}` | `backend-go/db/migrations/00023_condition_programs.sql:224` | |


## نوجوان (Teen)

محتوای مناسب سن برای نوجوان ۱۰ تا ۱۷ سال: نشانه‌ها، پرسش‌های متداول و کیف اولین پریود، و سیاست‌های حریم‌خصوصی/والد. برخی عبارت‌های «مادرت یا پزشک» باید بازنویسی شوند.

### الف) قواعد، آستانه‌ها و متن‌های داخل کد

| شناسه | چه چیزی باید بررسی شود | مقدار / متن فعلی (عین کد) | منبع | تصمیم بالینی |
|---|---|---|---|---|
| TN-C01 | گروه‌های سنی و پاسخ‌های «پریود شده‌ای؟» | سن: `10_12`, `13_15`, `16_17`؛ پریود: `not_yet`, `under_1y`, `over_1y` | `backend-go/internal/teen/teen.go` | |
| TN-C02 | داده‌ای که والد می‌بیند: فقط سطل هفتهٔ پریود بعدی (هرگز تاریخ/روز چرخه/فاز/تأخیر)؛ طول یادداشت برای والد | `this_week / next_week / later / unknown`؛ `MaxParentNote = 280` | `backend-go/internal/teen/teen.go` | |
| TN-C03 | سیاست‌ها: محتوای تجاری در حالت نوجوان نمایش داده نمی‌شود؛ ساخت کد پزشک/لینک گزارش برای نوجوان فقط با پیوند والد فعال (تصمیم ۱۴۰۵/۰۷ — ۲۰۲۶-۱۰-۱۲) | `AllowsCommercial(mode)`؛ `Sharing(...)` | `backend-go/internal/teen/policy.go` | |
| TN-C04 | عبارت «مادرت یا پزشک» در متن‌های رابط | «کی با مادرت یا پزشک صحبت کنی»؛ «شماره موبایل مادرت (یا بزرگ‌تری که به او اعتماد داری) را بنویس.…» | `frontend/messages/fa/teen.json` | |

### ب) ردیف‌های کاتالوگ (جدول catalog_items در seed)

همهٔ ردیف‌ها با `needs_review = 1` ذخیره شده‌اند. مقادیر زیر متن فارسی seed هستند (نسخهٔ انگلیسی در همان فایل است). ادمین بعد از تأیید می‌تواند پرچم را بردارد.

| شناسه | گروه / کد | چه چیزی باید بررسی شود | مقدار / متن فعلی | منبع | تصمیم بالینی |
|---|---|---|---|---|---|
| TN-K01 | `teen_signs` / `approaching_signs` | نشانه‌ها، برآورد و زمان صحبت با بزرگ‌تر/پزشک | **عنوان:** «نشانه‌های نزدیک شدن اولین پریود»<br>**متن:** «معمولاً حدود ۲ سال بعد از شروع رشد سینه‌ها. ترشح سفید یا شفاف از چند ماه قبل هم طبیعی است.»<br>**meta:** `{"kind": "sign", "menarche": ["not_yet"]}`<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:49` | |
| TN-K02 | `teen_signs` / `growth_spurt` | نشانه‌ها، برآورد و زمان صحبت با بزرگ‌تر/پزشک | **عنوان:** «جهش قد»<br>**متن:** «رشد ناگهانی قد و رویش مو زیر بغل معمولاً قبل از اولین پریود شروع می‌شود.»<br>**meta:** `{"kind": "sign", "menarche": ["not_yet"]}`<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:54` | |
| TN-K03 | `teen_signs` / `estimate_year_or_two` | نشانه‌ها، برآورد و زمان صحبت با بزرگ‌تر/پزشک | **عنوان:** «بر اساس جواب‌هایت: احتمالاً در یکی دو سال آینده»<br>**متن:** «هر بدنی زمان خودش را دارد. وقتی کیف اضطراری آماده باشد، غافلگیر نمی‌شوی.»<br>**meta:** `{"kind": "estimate", "menarche": ["not_yet"], "age_bands": ["10_12"]}`<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:59` | |
| TN-K04 | `teen_signs` / `estimate_coming_months` | نشانه‌ها، برآورد و زمان صحبت با بزرگ‌تر/پزشک | **عنوان:** «بر اساس جواب‌هایت: احتمالاً در ماه‌های آینده»<br>**متن:** «هر بدنی زمان خودش را دارد. وقتی کیف اضطراری آماده باشد، غافلگیر نمی‌شوی.»<br>**meta:** `{"kind": "estimate", "menarche": ["not_yet"], "age_bands": ["13_15"]}`<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:64` | |
| TN-K05 | `teen_signs` / `estimate_talk` | نشانه‌ها، برآورد و زمان صحبت با بزرگ‌تر/پزشک | **عنوان:** «بهتر است با مادرت یا پزشک صحبت کنی»<br>**متن:** «بیشتر دخترها تا ۱۵ سالگی پریود می‌شوند. صحبت با پزشک کمک می‌کند مطمئن شوی همه چیز روبه‌راه است.»<br>**meta:** `{"kind": "estimate", "menarche": ["not_yet"], "age_bands": ["16_17"], "severity": "caution"}`<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:69` | |
| TN-K06 | `teen_signs` / `estimate_first_year` | نشانه‌ها، برآورد و زمان صحبت با بزرگ‌تر/پزشک | **عنوان:** «سال اول: نامنظم بودن طبیعی است»<br>**متن:** «در ۱ تا ۲ سال اول فاصله پریودها ممکن است خیلی فرق کند. ثبت کردن کمک می‌کند الگوی خودت را ببینی.»<br>**meta:** `{"kind": "estimate", "menarche": ["under_1y"]}`<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:74` | |
| TN-K07 | `teen_signs` / `estimate_settling` | نشانه‌ها، برآورد و زمان صحبت با بزرگ‌تر/پزشک | **عنوان:** «چرخه‌ات در حال منظم شدن است»<br>**متن:** «ثبت کردن کمک می‌کند زمان پریود بعدی را زودتر بدانی و کیفت آماده باشد.»<br>**meta:** `{"kind": "estimate", "menarche": ["over_1y"]}`<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:79` | |
| TN-K08 | `teen_signs` / `when_to_talk` | نشانه‌ها، برآورد و زمان صحبت با بزرگ‌تر/پزشک | **عنوان:** «کی با مادرت یا پزشک صحبت کنی»<br>**متن:** «اگر تا ۱۵ سالگی پریود نشدی، یا درد و خونریزی خیلی زیاد داری، با مادرت یا پزشک صحبت کن.»<br>**meta:** `{"kind": "talk"}`<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:84` | |
| TN-K09 | `teen_faq` / `irregular` | پاسخ سؤال‌های متداول نوجوان | **عنوان:** «پریودم نامنظم است، مشکلی دارد؟»<br>**متن:** «در ۱ تا ۲ سال اول نامنظم بودن خیلی شایع و طبیعی است.»<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:89` | |
| TN-K10 | `teen_faq` / `how_much_bleeding` | پاسخ سؤال‌های متداول نوجوان | **عنوان:** «چقدر خونریزی طبیعی است؟»<br>**متن:** «معمولاً ۳ تا ۷ روز. اگر هر ساعت نوار پر می‌شود، به مادرت یا پزشک بگو.»<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:93` | |
| TN-K11 | `teen_faq` / `period_pain` | پاسخ سؤال‌های متداول نوجوان | **عنوان:** «درد پریود چه کنم؟»<br>**متن:** «گرما روی شکم، کمی تحرک و مسکن ساده با اجازه بزرگ‌ترها کمک می‌کند.»<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:97` | |
| TN-K12 | `teen_faq` / `pad_change` | پاسخ سؤال‌های متداول نوجوان | **عنوان:** «هر چند وقت نوار را عوض کنم؟»<br>**متن:** «معمولاً هر ۴ تا ۶ ساعت، یا زودتر اگر پر شده است.»<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:101` | |
| TN-K13 | `teen_faq` / `sports` | پاسخ سؤال‌های متداول نوجوان | **عنوان:** «در پریود می‌توانم ورزش کنم؟»<br>**متن:** «بله. ورزش سبک حتی ممکن است درد را کمتر کند.»<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:105` | |
| TN-K14 | `teen_kit_items` / `pads` | اقلام کیف اولین پریود | **عنوان:** «۲ نوار بهداشتی»<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:109` | |
| TN-K15 | `teen_kit_items` / `underwear` | اقلام کیف اولین پریود | **عنوان:** «یک لباس زیر اضافه»<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:112` | |
| TN-K16 | `teen_kit_items` / `wipes` | اقلام کیف اولین پریود | **عنوان:** «دستمال مرطوب»<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:115` | |
| TN-K17 | `teen_kit_items` / `pouch` | اقلام کیف اولین پریود | **عنوان:** «کیسه کوچک»<br>audiences: teen | `backend-go/db/migrations/00029_teen.sql:118` | |

### ج) بازنویسی‌های پیشنهادی QA (اعمال‌نشده)

| شناسه | ردیف | مقدار فعلی | پیشنهاد QA | منبع | تصمیم بالینی |
|---|---|---|---|---|---|
| TN-Q01 | `teen_signs/when_to_talk (عنوان و متن)` | «… با مادرت یا پزشک …» | «… با یک بزرگ‌تر مورد اعتماد (مثل مادرت) یا پزشک …» — همه مادر ندارند | `docs/qa/canvas/teen.md:91` | |
| TN-Q02 | `teen_signs/estimate_talk (عنوان)` | «بهتر است با مادرت یا پزشک صحبت کنی» | «خوب است با یک بزرگ‌تر مورد اعتماد یا پزشک صحبت کنی» | `docs/qa/canvas/teen.md:91` | |
| TN-Q03 | `teen_signs/estimate_talk (متن)` | «بیشتر دخترها تا ۱۵ سالگی پریود می‌شوند. …» | «هر بدنی زمان خودش را دارد. یک سر زدن به پزشک کمک می‌کند خیالت راحت باشد که همه چیز روبه‌راه است.» — نگرانی از «چیزی در من اشتباه است» | `docs/qa/canvas/teen.md:91` | |
| TN-Q04 | `teen_faq/how_much_bleeding (متن)` | «… به مادرت یا پزشک بگو» | «… به یک بزرگ‌تر مورد اعتماد یا پزشک بگو» | `docs/qa/canvas/teen.md:91` | |
| TN-Q05 | `teen_faq/period_pain (متن)` | «… مسکن ساده با اجازه بزرگ‌ترها …» | بدون تغییر ولی نیازمند تأیید بالینی (توصیهٔ دارو برای زیر ۱۸ سال) | `docs/qa/canvas/teen.md:91` | |


## پرونده سلامت و اسناد (Record)

پرونده سلامت و اسناد: فهرست نسبت‌ها در سابقهٔ خانوادگی، حدود قابل‌قبول برای به‌روزرسانی سن بارداری از سونو، و فیلدهای استخراج هوشمند.

### الف) قواعد، آستانه‌ها و متن‌های داخل کد

| شناسه | چه چیزی باید بررسی شود | مقدار / متن فعلی (عین کد) | منبع | تصمیم بالینی |
|---|---|---|---|---|
| RC-C01 | فهرست نسبت‌های سابقهٔ خانوادگی (۱۰ مورد) و برچسب‌ها | `mother, father, sister, brother, grandmother, grandfather, aunt, uncle, child, other` = مادر، پدر، خواهر، برادر، مادربزرگ، پدربزرگ، «خاله / عمه»، «دایی / عمو»، فرزند، دیگر | `backend-go/internal/healthrecord/extras.go (Relatives) ; backend-go/resources/translations/fa/record.json (extras.relatives.*)` | |
| RC-C02 | اجازهٔ به‌روزرسانی سن بارداری از سونوگرافی: حدود قابل‌قبول | سن بارداری هنگام سونو ۴ تا ۴۳ هفته (`minScanGA = 4*7`, `maxScanGA = 43*7-1`)؛ مدت بارداری `termDays = 280`؛ اختلاف > ۴۲ روز با زایمان فعلی = بارداری دیگر (`maxDatingDrift`)؛ زایمان گذشته‌تر از ۲۸ روز = بارداری قبلی؛ ga_days ۰ تا ۶، ga_weeks ۰ تا ۶۰ | `backend-go/internal/healthrecord/extract/dating.go` | |
| RC-C03 | متن پیشنهاد به‌روزرسانی سن بارداری | «این سونوگرافی سن بارداری را :weeks هفته و :days روز در تاریخ :date نوشته. سن بارداری‌ات را از همین سند به‌روز کنم؟» / «به نظر می‌رسد این سونوگرافی مربوط به بارداری دیگری است» | `backend-go/internal/healthrecord/extract/lang/fa/*.json (offer.*)` | |
| RC-C04 | فیلدهای استخراج‌شده از سند به‌ازای نوع (تصویربرداری: kind، ga_weeks، ga_days، edd، findings؛ ویزیت؛ نسخه + اقلام دارو؛ بستری؛ سایر) و انواع تصویربرداری | `ultrasound, xray, mri, ct, mammography, bone_density, other`؛ ویزیت: specialty/reason/diagnosis/next_visit؛ نسخه: medicine/dose/frequency/duration؛ بستری: date/ended_on/procedures | `backend-go/internal/healthrecord/extract/schema.go` | |
| RC-C05 | کارت اضطراری: نمایش آلرژی‌ها به‌طور پیش‌فرض روشن | `health_records.allergies_on_emergency_card` پیش‌فرض ۱ | `backend-go/db/migrations/00041_record_documents.sql` | |


## ثبت صوتی (Voice)

ثبت صوتی علائم: مقادیر پیش‌فرضی که وقتی کاربر چیزی نگفته جایگزین می‌شوند و نکتهٔ ایمنی صفحهٔ مرور.

### الف) قواعد، آستانه‌ها و متن‌های داخل کد

| شناسه | چه چیزی باید بررسی شود | مقدار / متن فعلی (عین کد) | منبع | تصمیم بالینی |
|---|---|---|---|---|
| VC-C01 | طول پیش‌فرض گرگرفتگی گفته‌شده بدون مدت (ثبت صوتی تایمر ندارد) | `DefaultFlashSeconds = 3 * 60` (۳ دقیقه) | `backend-go/internal/voicelog/canvas.go` | |
| VC-C02 | ساعت ثبت گرگرفتگی بدون ساعت: «دیشب» ۰۳:۰۰، وگرنه ظهر؛ سقف تعداد/مدت در یک جمله | `nightFlashHour = 3`، `dayFlashHour = 12`؛ `MaxVoiceFlashes = 10`؛ `MaxFlashMinutes = 60`؛ بازهٔ ثبت `MaxCommitDays = 7` | `backend-go/internal/voicelog/canvas.go` | |
| VC-C03 | «یک نکته» در صفحهٔ مرور ثبت صوتی | «یک نکته» / «خونریزی زیاد همراه با درد شدید را اگر ادامه داشت یا سرگیجه آوردی، به پزشک بگو.» | `frontend/messages/fa/voice-log.json (review.noteTitle, review.noteBody)` | |
| VC-C04 | سقف بیدارشدن شبانه برای ادرار از ثبت صوتی (هم‌سان با دفترچهٔ مثانه) | `MaxNightVoids = 20` | `backend-go/internal/voicelog/canvas.go` | |

