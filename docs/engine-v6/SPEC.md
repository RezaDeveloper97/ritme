<!-- Source of truth for milestone M9 (Engine v6). Converted 2026-10-21 from "Ritme_Spec_v6_Unified.docx" (user-supplied). -->
<!-- Do not edit rules here without the user OK: if code and this file disagree, this file wins (spec §0). -->

Ritme — Unified Product & Logic Spec v6
سند واحد ریتمی: جرنی‌ها، موتور چرخه، منطق ماژول‌های سلامت و مدل درآمد
این سند جایگزین کامل «Ritme Cycle Engine v1.1»، «Ritme Logic Spec v6» و «سند تحلیل فنی ریتمی» است و به هیچ سند دیگری ارجاع ندارد.
## ۰. راهنمای سند
- مرجع واحد: هر قاعده در این سند فقط یک بار آمده است. اگر پیاده‌سازی با این سند تناقض داشت، این سند مرجع است.
- بخش ۱ محصول و جریان‌ها · بخش ۲ موتور چرخه · بخش ۳ منطق ماژول‌های سلامت · بخش ۴ تحلیل، رضایت و مدل درآمد · بخش ۵ پنل ادمین · بخش ۶ تست‌ها و منابع.
- اصل ۱، تشخیص نه: هیچ خروجی قاعده‌محور یا هوش مصنوعی برچسب تشخیصی نمی‌دهد؛ فقط «در بازه / خارج از بازه / نیاز به پیگیری با پزشک» به‌علاوه منبع.
- اصل ۲، آستانه‌ها از منبع: همه عددها از رهنمودهای §۶.۲ هستند و در کد به‌صورت پیکربندی سمت سرور با ارجاع به منبع نگه داشته می‌شوند. تأیید نهایی با مشاور پزشکی ریتمی است.
- اصل ۳، داده کم، ادعا کم: هر تحلیل حداقل داده دارد (§۴.۱)؛ کمتر از آن «داده کافی نیست».
- اصل ۴، رضایت صریح: ارسال داده به هوش مصنوعی، پزشک یا همراه فقط با رضایت جداگانه و قابل لغو (§۴.۲).
- مرجع طراحی: کانواس Claude Design، دو پوسته Night & Bloom؛ پیشوند nbd_ تیره، nbl_ روشن، nb2_ و v19_ برای اقدام به بارداری و بارداری.
### تغییرات نسبت به Cycle Engine v1.1 (برای بازبینی انسانی)

| بخش | تغییر | دلیل |
|---|---|---|
| §۲.۱۸ پنجره زیستی | O−5 تا O+1 ← O−5 تا O | Wilcox 1995؛ رفع ناسازگاری با post_ovulation = low |
| §۲.۹ مقادیر موثر | قاعده گرد کردن میانه | میانه دو عدد اعشاری می‌شود |
| §۲.۳ تاریخ | میلادی و روز محلی؛ شمسی فقط نمایش | خطای طول ماه در مثال API |
| §۲.۳۵ مثال API | اصلاح ovulation، days_to_ovulation، days_to_period، subphase | تیر ۳۱ روز است |
| §۲.۱۶ و §۲.۱۷ | زنجیره ۶ سیکل آینده؛ اولویت BBT > LH > تقویم | تقویم آینده و حالت اقدام به بارداری |
| §۲.۲۷ | cycle_regularity و clinical_flags (FIGO) | برچسب منظم/نامنظم و پیام کاربر |
| §۲.۳۹ تا §۲.۴۳ | پریود رد شد، ویرایش گذشته، لنگر قدیمی، حالت‌های خاص، نگاشت UI | پوشش نداشت |
| §۱.۲ پیش‌فرض‌ها | جایگزینی پیش‌فرض با داده کاربر از اولین رکورد معتبر (نه بعد از ۲ سیکل) | هم‌راستایی با §۲.۹ |
| §۳.۱ تاریخ زایمان | اصلاح نگله بر اساس طول سیکل اختیاری، پیش‌فرض خاموش | ACOG CO 700 چنین اصلاحی را الزام نمی‌کند |
| §۳.۴ چکاپ‌ها | «خودآزمایی سینه» ← «آگاهی از سینه» | ACOG PB 179، USPSTF |
| §۳.۱ EPDS | فرم ۳ سؤالی طرح ← EPDS کامل ۱۰ سؤالی | ابزار معتبر |

## بخش ۱ — محصول، ناوبری و جریان‌ها
### ۱.۱ ناوبری و حالت‌ها
- تب‌بار: امروز · تقویم · + · تحلیل · من. «من» منوی اصلی همه ماژول‌های جانبی است: سلامت من، مراقبت و مشاوره، خانواده، آموزش، ابزارها، تنظیمات، پشتیبانی، درباره ریتمی.
- mode: cycle | ttc | pregnancy | postpartum؛ کاربر مرد: partner (فقط ریتمی همراه). تغییر حالت داده‌های قبلی را حذف نمی‌کند. محتوای «امروز»، شیت +، پیش‌بینی‌ها و تب تحلیل از روی mode ساخته می‌شوند.
### ۱.۲ اسپلش، معرفی و ثبت‌نام
اسپلش ← ۴ صفحه معرفی (سیکل؛ از اقدام تا مادری؛ سلامت در یک جا؛ حریم خصوصی) با «رد کردن» و «حساب دارم · ورود» ← ثبت‌نام:

| گام | محتوا | قاعده |
|---|---|---|
| ۱ | شماره موبایل ‎+98 | دو رضایت اجباری: قوانین و حریم خصوصی؛ پردازش داده سلامت |
| ۲ | OTP | ۵ رقم (قابل پیکربندی)، شمارش معکوس ارسال دوباره، خواندن خودکار پیامک |
| ۳ | جنسیت | مرد ← کد همراه ۶ کاراکتری ← اتصال یا «بعداً» · زن ← ادامه |
| ۴ | هدف | پیگیری سیکل \| اقدام به بارداری \| باردارم |
| ۵الف | سیکل (cycle و ttc) | تاریخ شروع آخرین پریود (یا «یادم نیست»)، طول پریود، طول سیکل |
| ۵ب | بارداری | مبنا: LMP \| سونوگرافی (تاریخ + هفته و روز) \| تاریخ زایمان \| هفته فعلی؛ نمایش هفته امروز و تاریخ تخمینی زایمان (§۳.۱) |
| ۶ | سلامت پایه | تاریخ تولد، قد، وزن (اختیاری) |
| ۷ | شرایط | بیماری مزمن، شرایط زنان، دارو یا روش پیشگیری (اختیاری) — روش هورمونی ورودی موتور است (§۲.۴۲) |
| ۸ | آماده | خلاصه پیش‌بینی اول + یادآور قبل از پریود |

- پیش‌فرض‌ها: اگر کاربر نمی‌داند، طول پریود ۵ و سیکل ۲۸ روز (declared یا default). از اولین رکورد معتبر واقعی، داده کاربر جایگزین می‌شود (§۲.۹).
- ریتمی همراه و خانواده: دسترسی اعضا به تفکیک دامنه و سطح در §۱.۷ تعریف شده است.
### ۱.۳ صفحه امروز (حالت سیکل)
ترتیب بلوک‌ها: هدر ← نوار ۷ روز ← حلقه سیکل (عدد کلیدی + دکمه وضعیت‌محور داخل حلقه) ← بنر وضعیت (شرطی، حداکثر یکی) ← کارت امروز (فاز، شانس بارداری، توضیح ۲ خطی، «بیشتر درباره این فاز» که شیت فاز را باز می‌کند) ← ثبت امروز ← پیش‌بینی‌ها (پریود بعدی، PMS، پنجره باروری، تخمک‌گذاری، طول سیکل + بج نظم) ← توصیه‌های امروز ← چالش امروز ← نکته هوشمند (لینک به تحلیل) ← کارت آموزش (فقط اگر کاربر دوره دارد). قواعد نمایش هر المان در §۲.۴۳.
### ۱.۴ ثبت داده (+)
- شیت + (ثبت سریع): با دکمه + باز می‌شود؛ ۸ کاشی سریع بر اساس mode و فاز، جست‌وجو، و همه دسته‌ها در سه گروه (بدن، ذهن و سبک زندگی، اندازه‌گیری). کاشی‌های پریود، درد و اندازه‌گیری شیت جزئیات کوچک خود را باز می‌کنند؛ بقیه صفحه ثبت کامل را روی همان بخش باز می‌کنند.
- صفحه ثبت کامل روز: همه دسته‌ها در یک اسکرول؛ از دکمه «ثبت کامل روز»، کارت «ثبت امروز»، «ثبت جزئیات» در تقویم، و ویرایش روزهای گذشته باز می‌شود.

| گروه | دسته | پارامترها |
|---|---|---|
| بدن | پریود و لکه‌بینی | شدت: لکه‌بینی / کم / متوسط / زیاد / خیلی زیاد (ثبت شدت، شروع پریود را ثبت می‌کند؛ لکه‌بینی نه) · جزئیات اختیاری: رنگ، لخته، بو |
| بدن | درد | محل: شکم، لگن، یک‌طرفه پایین، کمر، سر، سینه · شدت کم/متوسط/شدید (یا ۱ تا ۱۰ در نقشه بدن) |
| بدن | علائم بدن | گوارش: تهوع، نفخ، اسهال، یبوست · عمومی: خستگی، سرگیجه، گرگرفتگی، لرز، ورم · پوست و مو: جوش، پوست چرب، ریزش مو |
| بدن | ترشحات | قوام: ندارم، چسبنده، کرمی، آبکی، سفیده تخم‌مرغی · مقدار · رنگ و بو در صورت تغییر |
| بدن | ادراری و تناسلی | تکرر، سوزش ادرار، تغییر میزان ادرار، خشکی، سوزش و خارش واژن، تغییر بو |
| ذهن و سبک زندگی | حال | شاد، آرام، پرانرژی، حساس، زودرنج، مضطرب، غمگین، عصبانی، کلافه، بی‌حوصله |
| ذهن و سبک زندگی | اشتها و انرژی | انرژی ۳ سطح · اشتها کمتر/بیشتر · هوس شیرینی/شور |
| ذهن و سبک زندگی | خواب | مدت: ۰–۳، ۳–۶، ۶–۹، ۹+ ساعت · کیفیت ۴ سطح |
| ذهن و سبک زندگی | فعالیت بدنی | نوع، مدت، شدت |
| ذهن و سبک زندگی | رابطه و میل | میل ۳ سطح · رابطه: نداشتم / با محافظت / بدون محافظت · حین و بعد: خشکی، سوزش، درد، لکه‌بینی |
| اندازه‌گیری | وزن، دمای پایه، تست‌ها | kg · °C · LH (منفی/کم‌رنگ/مثبت) · تست بارداری — LH و BBT ورودی موتور‌اند (§۲.۱۷) |
| سایر | دارو و مکمل، یادداشت | از یادآورها · متن آزاد |

- بارداری: حرکات جنین، انقباض، فشار خون، علائم بارداری؛ خون‌ریزی به کارت هشدار «با پزشک در میان بگذار» تبدیل می‌شود. بعد از زایمان: خون‌ریزی بعد از زایمان، درد بخیه، سینه‌ها، شیردهی (سینه چپ/راست، شیشه، پمپ)، خواب و پوشک نوزاد، EPDS.
- شخصی‌سازی: پنهان کردن، سنجاق به ثبت سریع (حداکثر ۸)، ترتیب، مورد سفارشی.
### ۱.۵ نقشه ماژول‌ها

| ماژول | صفحه‌ها (کانواس) | منطق |
|---|---|---|
| اسپلش و ثبت‌نام | Splash، Intro_*، Onb_* | §۱.۲ |
| خانه سیکل | Cycle_Home، _Near، _During، Cycle_Phase | §۱.۳، §۲.۴۳ |
| ثبت (+) | Log_* | §۱.۴ |
| اقدام به بارداری / بارداری | nb2_TTC_*، nb2_Preg_* | §۲.۱۷، §۳.۱ |
| بعد از زایمان / فرزند | v15_*، v16_* | §۳.۱، §۳.۵ |
| یادآور دارو و نوبت | v13_* | زمان‌بندی محلی، اعلان خنثی |
| چکاپ‌ها | v14_* | §۳.۴ |
| علائم حیاتی | Vitals_* | §۳.۲ |
| تحلیل آزمایش | Lab_* | §۳.۳ |
| پرونده و گزارش پزشک | Record_* | PDF با انتخاب بخش‌ها، لینک ۷ روزه |
| تله‌مدیسین | v17_* | اشتراک داده با پزشک فقط با رضایت |
| دستیار سلامت | v18_* | سطح توصیه چهارپله‌ای؛ ارجاع به پزشک؛ بدون تشخیص |
| آموزش | LearnEntry_*، Learn_*، Ins_* | §۱.۶ |
| تحلیل | An_* | §۴.۱ |
| لیست کارها | Todo_* | مستقل؛ پیشنهاد هوشمند از سیکل |
| اشتراک | Prem_* | §۴.۳ |
| من و تنظیمات | Me_* | منوی مرکزی، حریم خصوصی، حذف داده |

### ۱.۶ آموزش (کاربر و مدرس)
- دسترسی: مدرس شماره موبایل را به دوره، گروه یا محتوای مستقل وصل می‌کند و مدت (نامحدود / N روز / تا تاریخ) تعیین می‌کند. شماره عضو ← دسترسی فوری؛ غیرعضو ← pending و فعال‌سازی خودکار هنگام ثبت‌نام با همان شماره. پایان دسترسی ← حذف فایل‌های دانلودشده.
- نقطه ورود کاربر: کارت «ادامه یادگیری» یا «دوره جدید» در صفحه امروز (فقط اگر دوره دارد)، منوی من ← دوره‌های من، و اعلان + شیت «دسترسی جدید». تب جدا فقط اگر آموزش هسته محصول شد.
- درس‌ها: ویدیو، صوت (پخش پس‌زمینه، تایمر خواب، سرعت)، PDF؛ پیشرفت، «این درس را تمام کردم»، دانلود آفلاین فقط داخل اپ.
### ۱.۷ ریتمی همراه و خانواده
«همراه» از اشتراک فاز چرخه به یک حلقه خانواده گسترش یافته است: صاحب داده (زن) اعضایی را اضافه می‌کند و برای هر عضو، دسترسی را به تفکیک نوع داده تعیین می‌کند. محتوای عضو بر اساس mode صاحب داده (چرخه، بارداری، بعد از زایمان) و پروفایل کودکان ساخته می‌شود. رفرنس‌ها: Apple Health Sharing (فقط خلاصه، قابل توقف)، Flo for Partners (عضو نمی‌تواند ثبت کند)، GoHenry (والد اصلی و والد مشترک)، Amazon Family (بزرگسال با حساب در برابر پروفایل کودک).

| سطح | عضو چه می‌تواند بکند |
|---|---|
| خاموش | هیچ |
| مشاهده | فقط خلاصه را می‌بیند |
| یادآوری | مشاهده + ارسال یادآوری ملایم به صاحب داده (حداکثر ۱ بار برای هر دوز یا نوبت، ۳ بار در روز) |
| ثبت | مشاهده + افزودن یا ثبت؛ برای داده‌های خود صاحب، موارد جدید (دارو، نوبت) پیشنهاد می‌شوند و بعد از تأیید او اعمال می‌شوند |

| دامنه داده | حداکثر سطح مجاز | پیش‌فرض همسر/پارتنر | پیش‌فرض مراقب |
|---|---|---|---|
| فاز کلی چرخه | مشاهده | مشاهده | خاموش |
| هفته و تاریخ تقریبی زایمان | مشاهده | مشاهده | مشاهده |
| ویزیت‌ها و نوبت‌های صاحب | ثبت (با تأیید) | ثبت | مشاهده |
| حرکات جنین و انقباض (خلاصه) | مشاهده | مشاهده | خاموش |
| وزن، فشار، قند صاحب | مشاهده | خاموش | خاموش |
| دارو و مکمل صاحب | یادآوری؛ افزودن فقط با تأیید | یادآوری | خاموش |
| هشدار فوری (پیگیری زودتر) | دریافت عنوان و راه تماس | روشن | خاموش |
| پروفایل کودک (واکسن، رشد، شیر، خواب، دارو) | ثبت | ثبت (والد مشترک) | ثبت محدود: شیر، پوشک، خواب |
| لیست کارهای خانه | ثبت | ثبت | خاموش |

- هرگز به اشتراک گذاشته نمی‌شود: یادداشت‌های خصوصی، حال روحی و نمره EPDS، رابطه و میل، جواب آزمایش‌ها، گفت‌وگو با دستیار و پزشک، تاریخچه کامل چرخه، و داده‌های باروری (مگر نقش همسر در حالت اقدام به بارداری و با سوییچ جداگانه).
- نقش‌ها و قالب پیش‌فرض: همسر یا پارتنر · والد دیگر کودک (فقط کودکان مشترک) · مراقب یا خانواده (پیش‌فرض زمان‌دار، مثلاً تا ۶ هفته بعد از زایمان) · دوست یا همراه چرخه (فقط فاز، همان ریتمی همراه). نقش فقط پیش‌فرض را می‌چیند؛ هر ردیف جداگانه قابل تغییر است.
- فعال‌سازی بر اساس mode: دامنه‌های بارداری فقط وقتی صاحب در mode = pregnancy است دیده می‌شوند؛ با تغییر mode دسترسی‌های همان دامنه خودکار غیرفعال (نه حذف) می‌شوند و پیشنهاد تنظیم دامنه‌های جدید به صاحب نمایش داده می‌شود.
- مالکیت داده کودک: پروفایل کودک مالک مشترک دارد (والدین با سطح «ثبت»). هر ثبت با شناسه ثبت‌کننده ذخیره می‌شود و در «فعالیت اعضا» تا ۲۴ ساعت قابل برگرداندن است.
- هشدار فوری: فقط پیام‌های alerts_inbox با level = پیگیری زودتر و فقط اگر صاحب سوییچ «هشدار فوری» را برای آن عضو روشن کرده باشد؛ محتوا فقط عنوان، زمان و دکمه تماس با صاحب و ۱۱۵ (بدون جزئیات پزشکی).
- لغو و ممیزی: حذف عضو یا خاموش کردن دامنه فوراً اعمال می‌شود و عضو دلیل آن را نمی‌بیند. همه دسترسی‌ها، یادآوری‌ها، پیشنهادها و ثبت‌ها در جدول family_activity ثبت می‌شوند.
```
family_member { owner_id, member_user_id, role, status: invited|active|revoked, expires_at }
family_grant  { member_id, domain, level: off|view|nudge|write, updated_at }   // one row per domain
child_profile { id, guardians[{user_id, level}] }
family_activity { actor_id, subject_id, domain, action, payload_ref, created_at, undone_at }
check(actor, subject, domain, action) -> allow | propose(needs owner approval) | deny
```

یک فهرست برای همه («همراهان و خانواده» در تب من): همدم و خانواده دو محصول جدا نیستند؛ هر نفر یک «پیوند اشتراک» جهت‌دار است و فرق همدم با همسر فقط در grantهاست. اشتراک همیشه یک‌طرفه تعریف می‌شود (مالک → عضو)؛ دوطرفه یعنی دو پیوند مستقل که هر طرف پیوند خودش را کنترل می‌کند.

| شروع سریع (template) | grantها | کاربرد |
|---|---|---|
| همدم چرخه | فاز کلی: مشاهده | دوست یا همسر فقط برای چرخه (همان ریتمی همراه) |
| نوبت و دارو | نوبت‌ها: مشاهده و ثبت با تأیید · دارو: مشاهده و یادآوری · چرخه: خاموش | زن و شوهری که فقط کارهای درمانی هم را دنبال می‌کنند |
| مراقبت کامل خانواده | بارداری، نوبت‌ها، دارو، هشدار فوری، کودکان | بارداری و بعد از زایمان |
| سفارشی | هر ترکیب | — |

- جریان افزودن نفر: نسبت (همسر، دوست، خانواده، مراقب) ← شروع سریع + ویرایش ردیف‌ها ← «از او هم درخواست کن» (اختیاری، برای نوبت‌ها و داروهای خود او) ← مرور، رضایت و دعوت با شماره یا کد.
- پذیرش در سمت عضو: عضو خلاصه آنچه می‌بیند را مرور می‌کند و اگر درخواست متقابل آمده باشد، برای نوبت‌ها و داروهای خودش سطح انتخاب می‌کند؛ می‌تواند بدون اشتراک متقابل هم قبول کند.
- درخواست دسترسی بیشتر: هر طرف می‌تواند از طرف دیگر دسترسی بیشتری درخواست کند (share_request)؛ تا تأیید مالک هیچ تغییری اعمال نمی‌شود.
- دامنه‌های مرد: اگر کاربر مرد نوبت یا دارو داشته باشد، خودش مالک آن‌هاست و با همان مدل (مالک → عضو) با همسرش به اشتراک می‌گذارد. صفحه «دارو و نوبت‌های ما» برای هر طرف فقط موارد مجاز را نشان می‌دهد.
```
share_link    { owner_id, member_id, relation, template, status, expires_at }   // directional
share_request { from_id, to_id, domains[], requested_level, status: pending|accepted|declined }
mutual = share_link(A->B) + share_link(B->A); revoking one never touches the other
```

### ۱.۸ خانواده: جریان‌ها و ویژگی‌ها

| صفحه (کانواس) | کاربر | ویژگی‌ها |
|---|---|---|
| Fam_Hub | مدیر حلقه | اعضا با خلاصه دسترسی، پروفایل کودکان، موارد مشترک (یادآورها، نوبت‌ها، لیست کارها)، ورود به فعالیت اعضا |
| Fam_AddMember | مدیر حلقه | انتخاب نقش با قالب پیش‌فرض؛ دعوت بزرگسال با کد یا شماره؛ افزودن پروفایل کودک |
| Fam_Access | مدیر حلقه | ماتریس دامنه × سطح با سطوح غیرمجاز غیرفعال، سوییچ هشدار فوری، مدت دسترسی، فهرست «هرگز»، حذف عضو |
| Fam_Activity | مدیر حلقه | لاگ فعالیت با فیلتر عضو، برگرداندن تا ۲۴ ساعت، تأیید یا رد پیشنهادها |
| Fam_PHome | عضو | خانه بر اساس mode صاحب: خلاصه بارداری، کارهای امروز خانواده، نوبت‌ها با «من هم می‌آیم»، پیشنهاد حمایت، لیست کارها |
| Fam_PMeds | عضو | داروها به تفکیک فرد با اقدام متناسب سطح: یادآوری ملایم، ثبت دوز، پیشنهاد دارو، یادآور شخصی |
| Fam_PCalendar | عضو | تقویم مشترک با فیلتر؛ حضور در نوبت؛ افزودن نوبت (مستقیم برای کودک، با تأیید برای صاحب، شخصی برای خودش) |
| Fam_PChild | عضو (والد مشترک) | ثبت سریع شیر، پوشک، خواب و دارو؛ واکسن، رشد، مراحل؛ نام ثبت‌کننده روی هر ثبت |
| Fam_PAlert | عضو | هشدار «پیگیری زودتر» صاحب: عنوان، زمان، تماس با صاحب و ۱۱۵، راهنمای کمک، تاریخچه |
| Fam_PPostpartum | عضو | شیفت شب، شیر دوشیده، ثبت شیر، ویزیت بعد از زایمان، کارت عمومی «حالش را بپرس» |
| Comp_* (نقش دوست) | هر دو | همراه چرخه: فقط فاز کلی و اشاره‌های مجاز با زبان احتمالی؛ پنل شخصی مرد بدون اتصال |
| Me_People، People_* | هر دو | نقطه ورود در تب من؛ افزودن نفر، شروع سریع، اشتراک متقابل، پذیرش، دارو و نوبت‌های ما، تنظیمات دوطرفه |

جریان دعوت:
```
owner: choose role -> template grants -> (optional) edit grants -> create invite
invite { code: 6 chars, single-use, expires 24h } or { phone -> auto-bind on signup }
member: enter code -> sees role + granted domains (read-only summary) -> accept
status: invited -> active ; expired/declined -> removed ; owner can cancel any time
```

بررسی دسترسی و تأیید:
```
check(actor, subject, domain, action):
  if actor == subject.owner: allow
  grant = family_grant(actor, domain); if none or expired or mode-inactive: deny
  view   -> allow if grant >= view
  nudge  -> allow if grant >= nudge and nudge_quota_ok
  write  -> if subject is child_profile and guardian(actor): allow
            elif grant == write: create proposal(status=pending) -> owner approves/rejects
  every allow/propose is written to family_activity
```

- پیشنهاد (proposal): pending ← approved (اعمال و اعلان به عضو) یا rejected؛ اگر ۷ روز بی‌پاسخ بماند expired. پیشنهاد دارو هرگز بدون تأیید صاحب یادآور فعال نمی‌سازد.
- یادآوری ملایم (nudge): فقط وقتی دوز یا نوبت در موعد ثبت نشده (دارو: ۶۰ دقیقه بعد از موعد). حداکثر ۱ بار برای هر دوز یا نوبت، ۳ بار در روز برای هر عضو؛ ساعت سکوت ۲۳ تا ۸. متن ثابت و مهربان، بدون ذکر نام دارو روی صفحه قفل.
- اعلان‌ها به عضو: تغییر فاز یا هفته (خلاصه)، نوبت فردا، دوز ثبت‌نشده (فقط سطح یادآوری)، هشدار فوری (فقط با سوییچ). همه اعلان‌ها روی صفحه قفل خنثی هستند («پیامی از خانواده در ریتمی»).
- تغییر mode صاحب: دامنه‌های mode قبلی غیرفعال (نه حذف) می‌شوند؛ صاحب کارت «دسترسی‌های جدید را تنظیم کن» می‌بیند؛ عضو تا تأیید صاحب فقط دامنه‌های مشترک (کودکان، لیست کارها) را می‌بیند. زایمان: پیشنهاد خودکار ایجاد پروفایل کودک با والدین مشترک.
- کودک و والدین مشترک: هر دو والد «ثبت» دارند؛ تعارض هم‌زمان با آخرین ثبت و ماندن هر دو رکورد در لاگ حل می‌شود. حذف پروفایل کودک یا حذف یک والد از پروفایل نیاز به تأیید هر دو والد دارد. مراقب فقط دامنه‌های تعیین‌شده را دارد.
- خروج و حذف: عضو هر زمان می‌تواند خودش خارج شود؛ حذف حساب صاحب، همه grantها را فوراً باطل می‌کند و داده صاحب از دید اعضا حذف می‌شود؛ داده کودک برای والد دیگر باقی می‌ماند.
- ادغام با موتور پیام: هشدارهای فوری از alerts_inbox صاحب (§۵.۴.۱) با type = safety و level = پیگیری زودتر، اگر سوییچ روشن باشد، به slot جدید member_alert عضو کپی می‌شوند. پیشنهادهای حمایت در خانه عضو از slot member_tip با قاعده بر mode و هفته صاحب می‌آیند و هرگز به داده‌های «هرگز» شرط نمی‌کنند.
- پنل ادمین: در جزئیات کاربر، تب «خانواده» فقط تعداد اعضا، نقش‌ها و وضعیت دعوت‌ها را نشان می‌دهد (بدون محتوای grantها)؛ KPIهای §۵.۶ شامل نرخ پذیرش دعوت، میانگین اعضای هر حلقه، نرخ پذیرش پیشنهادها و یادآوری‌ها.
- اشتراک: همه امکانات خانواده برای صاحب و اعضا رایگان است؛ اگر صاحب پلاس داشته باشد، امکانات پلاس مرتبط با داده‌های مشترک (مثل گزارش PDF) برای عضو با سطح مشاهده هم در دسترس است.
## بخش ۲ — موتور چرخه
### ۲.۱ هدف
موتور وضعیت چرخه را برای یک target_date تعیین می‌کند، خروجی روزانه برای Frontend می‌دهد، پریود و پنجره باروری را پیش‌بینی می‌کند، داده واقعی را از تخمینی جدا نگه می‌دارد، مانع ورود پیش‌بینی به داده واقعی می‌شود، داده ناقص و edge caseها را مدیریت می‌کند و خروجی قابل توضیح برای UI و debug می‌دهد. موتور ابزار تخمینی است؛ تشخیص پزشکی یا روش پیشگیری از بارداری نیست.
### ۲.۲ تفکیک داده واقعی از محاسباتی
```
period_logs     = actual user-logged data only
engine_outputs  = computed, temporary
predictions     = estimated, never actual
```

- predicted_period_start هرگز در period_logs ذخیره نمی‌شود.
- assumed_period_end هرگز به‌عنوان period_end واقعی ذخیره نمی‌شود.
- داده onboarding یا fallback همیشه source مشخص دارد.
### ۲.۳ تاریخ هدف، تقویم و منطقه زمانی
- همه محاسبات برای target_date انجام می‌شوند (تاریخی که موتور باید وضعیت را برایش resolve کند).
- تاریخ‌ها در موتور از نوع Date میلادی بدون ساعت هستند. تبدیل به شمسی فقط در لایه نمایش با کتابخانه معتبر؛ هیچ محاسبه‌ای روی رشته شمسی یا با فرض ماه ثابت انجام نمی‌شود.
- «روز» = روز محلی کاربر بر اساس timezone دستگاه (پیش‌فرض Asia/Tehran)، نه UTC سرور.
### ۲.۴ اختلاف روزها
```
difference_in_days(date_a, date_b) = calendar days of (date_a - date_b)
example: difference_in_days(2026-07-01, 2026-06-22) = 9
```

### ۲.۵ روز چرخه
```
cycle_day = difference_in_days(target_date, current_period_start) + 1   // 1-based
if current_period_start is unknown: cycle_day = null
```

### ۲.۶ ورودی‌ها

| گروه | فیلدها | نوع |
|---|---|---|
| واقعی | period_start, period_end, cycle_history, bleeding_history | actual |
| onboarding / profile | declared_cycle_length, declared_period_length, last_period_start_from_onboarding, birth_date | declared — actual log نیست |
| نشانگرهای باروری | lh_tests[date, result], bbt[date, value], cervical_mucus[date, type] | actual |
| زمینه | mode, hormonal_contraception, breastfeeding, skipped_cycles, positive_pregnancy_test | profile / actual |

### ۲.۷ بازه پشتیبانی و outlier
```
cycle_length supported for calculation: 21..45
period_length typical: 2..8
cycle_length < 21 => short_cycle_outlier
cycle_length > 45 => long_cycle_outlier
```

outlierها فقط طبق قواعد این بخش در warning، confidence و data_quality اثر می‌گذارند و خام وارد prediction نمی‌شوند. بازه ۲۱ تا ۴۵ فیلتر محاسبه است، نه معیار «طبیعی»؛ معیار کاربر در §۲.۲۷.
### ۲.۸ رکورد معتبر
```
valid cycle length:  21 <= cycle_length <= 45 AND cycle not skipped
valid period length: 2 <= period_length <= 10
```

period_length ۲ تا ۸ = typical؛ ۹ تا ۱۰ = معتبر ولی کم‌رایج و بسته به history ممکن است confidence را کم کند؛ کمتر از ۲ یا بیشتر از ۱۰ = نامعتبر برای محاسبه.
### ۲.۹ مقادیر موثر
```
effective_cycle_length  = round_half_up(median(last_3_valid_cycle_lengths))
effective_period_length = round_half_up(median(last_3_valid_period_lengths))
3 records -> median of 3 | 2 -> mean of 2, round .5 up | 1 -> that value | 0 -> fallback
fallback cycle:  declared_cycle_length -> 28
fallback period: declared_period_length -> 5
example: median(28, 29) = 28.5 -> 29
```

داده predicted و assumed_period_end هرگز وارد این محاسبه نمی‌شوند.
### ۲.۱۰ منابع داده
```
current_period_start_source: user_logged | onboarding_declared | predicted_reference | unknown
current_period_end_source:   user_logged | assumed_from_effective_duration | unknown
ovulation_source:            bbt_confirmed | lh_positive | calendar | unknown
```

قانون اجباری: هر جا current_period_start یا current_period_end در خروجی است، source آن هم برمی‌گردد؛ و current_period_end همیشه با current_period_end_is_confirmed.
### ۲.۱۱ مدل پایان پریود
```
if actual period_end exists:
  current_period_end = period_end; source = user_logged; is_confirmed = true
else:
  assumed_period_end = current_period_start + effective_period_length - 1
  current_period_end = assumed_period_end; source = assumed_from_effective_duration; is_confirmed = false
```

### ۲.۱۲ سقف‌ها برای period_end ثبت‌نشده
```
period_soft_cap    = effective_period_length
period_warning_cap = max(effective_period_length + 3, 8)
period_hard_cap    = 12
```

| days_since_period_start | وضعیت |
|---|---|
| تا soft_cap | menstrual عادی |
| soft_cap + 1 تا warning_cap | menstrual_possible |
| warning_cap + 1 تا hard_cap | menstrual_possible با هشدار شدید و confidence پایین |
| بیشتر از hard_cap | menstrual فعال پایان‌یافته فرض می‌شود؛ سیستم فوراً unknown نمی‌شود و ادامه سیکل را resolve می‌کند |

حد ۸ در warning_cap با حد «پریود طولانی» FIGO (بیش از ۸ روز) هم‌راستاست.
### ۲.۱۳ main_phase
```
menstrual | follicular | fertile | luteal | period_expected | unknown
```

### ۲.۱۴ و ۲.۱۵ subphase و نگاشت

| main_phase | subphaseهای مجاز |
|---|---|
| menstrual | menstrual, menstrual_possible |
| follicular | early_follicular, mid_follicular, late_follicular_transition |
| fertile | fertile_rising, high_fertility, ovulation_likely |
| luteal | post_ovulation, early_luteal, mid_luteal, late_luteal, pms_possible |
| period_expected | period_expected |
| unknown | unknown |

### ۲.۱۶ پیش‌بینی پریود بعدی و سیکل‌های آینده
```
predicted_next_period_start = current_period_start + effective_cycle_length
// = cycle_day (effective_cycle_length + 1)
```

```
if ovulation_source in {bbt_confirmed, lh_positive} for the current cycle:
  predicted_next_period_start = ovulation_date + effective_luteal_length
  effective_luteal_length = median(BBT-confirmed luteal lengths, valid 10..16) else 14
```

```
future cycles (display only, never stored), k = 1..6:
  predicted_start[k] = predicted_next_period_start + (k-1) * effective_cycle_length
  period, fertile window and ovulation for each k use §2.17-2.19
  confidence for k >= 2 is at most medium
```

### ۲.۱۷ تخمک‌گذاری

| اولویت | منبع | قاعده |
|---|---|---|
| ۱ | bbt_confirmed | قاعده «سه بالای شش»: وقتی ۳ دمای متوالی از بیشینه ۶ دمای قبلی بالاتر باشد (حالت سخت‌گیرانه: دمای سوم ≥ ۰٫۲°C بالاتر)، تخمک‌گذاری = روز قبل از اولین دمای بالا. فقط بعد از وقوع تأیید می‌کند. حداقل ۱۰ دما در سیکل. |
| ۲ | lh_positive | estimated_ovulation_date = اولین روز LH مثبت + ۱ (تخمک‌گذاری معمولاً ۲۴ تا ۳۶ ساعت بعد از شروع افزایش LH) |
| ۳ | calendar | estimated_ovulation_date = predicted_next_period_start − ۱۴ (luteal_length ثابت در MVP) |

```
ovulation_cycle_day = difference_in_days(estimated_ovulation_date, current_period_start) + 1
if |ovulation(lh) - ovulation(bbt)| > 2 days: use BBT; warnings += lh_bbt_conflict
```

### ۲.۱۸ پنجره باروری زیستی
```
fertile_biological_start = estimated_ovulation_date - 5
fertile_biological_end   = estimated_ovulation_date      // O-5..O, 6 days (Wilcox 1995)
```

### ۲.۱۹ پنجره باروری نمایشی
```
display_fertile_start = max(fertile_biological_start, current_period_end + 1)
display_fertile_end   = estimated_ovulation_date
if display_fertile_end < display_fertile_start: window is empty (no day resolves to fertile)
menstrual always overrides fertile
```

### ۲.۲۰ اولویت resolve
۱) menstrual واقعی یا فعال با end فرضی · ۲) period_expected · ۳) پنجره fertile نمایشی · ۴) luteal · ۵) follicular · ۶) unknown. «داده ناقص» فاز مستقل نیست و از طریق warnings، data_quality، requires_user_input، confidence و confidence_reasons منعکس می‌شود.
### ۲.۲۱ منطق menstrual
```
days_since_period_start = difference_in_days(target_date, current_period_start) + 1
```

| حالت | شرط | خروجی |
|---|---|---|
| ۲۱.۱ | is_confirmed = true و start ≤ target ≤ end | menstrual / menstrual · resolution_source = user_logged |
| ۲۱.۲ | is_confirmed = false و days_since ≤ soft_cap | menstrual / menstrual · user_logged_with_assumed_end |
| ۲۱.۳ | soft_cap < days_since ≤ warning_cap | menstrual / menstrual_possible · warnings += period_end_missing · user_logged_with_assumed_end |
| ۲۱.۴ | warning_cap < days_since ≤ hard_cap | menstrual / menstrual_possible · + period_end_missing_warning_cap_exceeded · data_quality = poor · confidence = low · requires_user_input = true |
| ۲۱.۵ | days_since > hard_cap | menstrual فعال نیست؛ assumed_end فقط anchor داخلی؛ فاز از ادامه سیکل · + period_end_missing_hard_cap_exceeded · poor · requires_user_input · confidence_reasons += missing_period_end_beyond_hard_cap |

### ۲.۲۲ period_expected
```
condition: target_date >= predicted_next_period_start AND no new period_start on/before target_date
raw = difference_in_days(predicted_next_period_start, target_date)
days_to_period = max(0, raw); days_late = max(0, -raw)
```

| days_late | خروجی |
|---|---|
| ۰ تا ۷ | period_expected · resolution_source = prediction |
| ۸ تا ۱۴ | period_expected · warnings += predicted_period_overdue · confidence = low |
| بیشتر از ۱۴ | unknown · + predicted_period_overdue, cycle_unresolved_after_expected_period · requires_user_input · low · poor · anchors، ovulation و days_late همچنان برگردانده می‌شوند · days_to_period = 0 · days_to_ovulation = null · cycle_day فقط اگر start معتبر است |

main_phase = unknown به معنای invalid شدن کل خروجی نیست. با ثبت period_start واقعی برای target_date یا قبل از آن، سیکل تأییدشده جدید فوراً شروع می‌شود و period_expected دیگر اعمال نمی‌شود.
suggest_pregnancy_test = true وقتی mode ∈ {cycle, ttc}، hormonal_contraception = false، و days_late ≥ ۱ در ttc یا days_late ≥ ۸ در cycle.
### ۲.۲۳ follicular
```
follicular_start = current_period_end + 1
follicular_end   = display_fertile_start - 1
range_length     = difference_in_days(follicular_end, follicular_start) + 1
if follicular_end < follicular_start: no follicular range
range 1 day : mid_follicular
range 2 days: day1 early_follicular, day2 late_follicular_transition
range >= 3  : early = floor(range*0.4); mid = floor(range*0.4); late = range - early - mid
              if early == 0: early = 1; if mid == 0: mid = 1; remainder goes to late
```

late_follicular_transition در API باقی می‌ماند؛ UI می‌تواند آن را با mid_follicular یکی نمایش دهد.
### ۲.۲۴ fertile
```
only inside a non-empty display fertile window
days_to_ovulation = difference_in_days(estimated_ovulation_date, target_date)
5,4,3 -> fertile_rising | 2,1 -> high_fertility | 0 -> ovulation_likely
```

اگر روزی داخل پنجره زیستی باشد اما به‌خاطر menstrual override یا پنجره نمایشی خالی نمایش داده نشود، main_phase = fertile نمی‌شود.
### ۲.۲۵ luteal
```
applies when: estimated_ovulation_date < target_date < predicted_next_period_start
days_since_ovulation = difference_in_days(target_date, estimated_ovulation_date)
days_until_period    = difference_in_days(predicted_next_period_start, target_date)
priority: pms_possible > late_luteal > post_ovulation > early/mid
days_until_period 1..3        -> pms_possible
days_until_period 4..6        -> late_luteal
days_since_ovulation = 1      -> post_ovulation
assignable range: ovulation + 2 .. predicted_next_period_start - 7
  assignable_length = difference_in_days(end, start) + 1 (none if end < start)
  early_luteal = first ceil(length/2) days; mid_luteal = remaining floor(length/2) days
```

### ۲.۲۶ fertility_level

| subphase | fertility_level |
|---|---|
| menstrual، menstrual_possible، early_follicular، mid_follicular | low |
| late_follicular_transition، fertile_rising | medium |
| high_fertility | high |
| ovulation_likely | peak |
| post_ovulation، early_luteal، mid_luteal، late_luteal، pms_possible | low |
| period_expected، unknown، و هر حالتی در §۲.۴۲ که باروری سرکوب می‌شود | unknown |

### ۲.۲۷ تنوع، نظم و پرچم‌های بالینی
```
cycle_variability = max(last_3_valid) - min(last_3_valid)   // null if < 2 valid cycles; used for confidence only
```

cycle_regularity برای UI و گزارش: روی سیکل‌های کامل واقعی ۱۲ ماه اخیر (حداقل ۳، بدون skipped). اختلاف بلندترین و کوتاه‌ترین با آستانه FIGO: سن ۲۶ تا ۴۱ ← حداکثر ۷ روز؛ ۱۸ تا ۲۵ و ۴۲ تا ۴۵ ← حداکثر ۹ روز.
```
cycle_regularity = regular | irregular | insufficient_data
```

| clinical_flag | مقادیر | قاعده |
|---|---|---|
| frequency | frequent \| normal \| infrequent | effective_cycle_length کمتر از ۲۴ / ۲۴ تا ۳۸ / بیشتر از ۳۸ (FIGO) |
| duration | normal \| prolonged | period_length واقعی بیشتر از ۸ روز |
| absent | true \| false | ۹۰ روز بدون پریود و mode نه بارداری و نه بعد از زایمان |
| نوجوان | — | سن کمتر از ۱۸ یا ≤ ۳ سال از اولین پریود: بازه طبیعی ۲۱ تا ۴۵ (ACOG CO 651) |

### ۲.۲۸ data_quality
enum: good | partial | poor | insufficient · ترتیب بررسی: insufficient ← poor ← good ← partial.

| مقدار | شرط |
|---|---|
| insufficient | current_period_start وجود ندارد و onboarding یا fallback هم anchor کافی نمی‌دهد |
| poor | هرکدام: end تأییدنشده و days_since > warning_cap · days_late > 14 · anchorهای ناقص یا متناقض · فقط history outlier موجود |
| good | همه: start واقعی · anchorهای لازم کامل · حداقل ۲ رکورد معتبر واقعی · target در وضعیت unresolved نیست · days_late ≤ 7 |
| partial | سایر حالت‌های قابل resolve: anchor مکمل declared یا assumed · history معتبر کمتر از ۲ · onboarding یا fallback در محاسبه |

### ۲.۲۹ confidence
enum: high | medium | low | unknown · ترتیب بررسی: unknown ← low ← high ← medium.

| مقدار | شرط |
|---|---|
| unknown | resolution_source = unknown یا anchor کافی نیست |
| low | هرکدام: main_phase = unknown · days_late > 7 · data_quality = poor · end تأییدنشده و days_since > warning_cap · history معتبر کمتر از ۲ · cycle_variability > 5 · resolution_source ∈ {default_based, onboarding_based} · start از predicted_reference · hormonal_contraception = true · postpartum با کمتر از ۲ پریود واقعی |
| high | همه: data_quality = good · ≥ ۲ طول سیکل و ≥ ۲ طول پریود معتبر واقعی · start_source = user_logged · اگر end در resolve دخیل است، تأییدشده · cycle_variability ≤ 5 (یا ovulation_source = bbt_confirmed در سیکل جاری) · main_phase ∉ {period_expected, unknown} · resolution_source ∉ {default_based, onboarding_based, unknown} |
| medium | سایر حالت‌ها؛ برای future_cycles با k ≥ 2 حداکثر medium؛ سن ≥ ۴۵ حداکثر medium |

confidence_reasons (همه علت‌های فعال): low_history · high_cycle_variability · missing_period_end · missing_period_end_beyond_warning_cap · missing_period_end_beyond_hard_cap · onboarding_data_used · default_values_used · prediction_based_output · period_overdue · unresolved_cycle · poor_data_quality · insufficient_anchor_data · stale_onboarding_anchor · hormonal_contraception_active.
### ۲.۳۰ resolution_source
enum: user_logged | user_logged_with_assumed_end | prediction | onboarding_based | default_based | unknown. منبع اصلی لازم برای resolve وضعیت target_date را نشان می‌دهد؛ ترتیب enum فقط tie-breaker است.

| مقدار | وقتی |
|---|---|
| user_logged | start و end واقعی و فاز مستقیماً از آن‌ها resolve می‌شود (مثلاً داخل پریود واقعی) |
| user_logged_with_assumed_end | start واقعی، end فرضی، و فاز هنوز به menstrual فعال با end فرضی متکی است |
| prediction | فاز از predicted_next_period_start، ovulation، پنجره نمایشی، luteal یا period_expected می‌آید (حتی اگر start و end واقعی باشند) |
| onboarding_based | هیچ log واقعی برای سیکل جاری نیست و resolve بر اساس anchor آنبوردینگ است |
| default_based | نه log کافی نه onboarding کافی؛ مقادیر پیش‌فرض |
| unknown | حتی با fallback anchor کافی نیست |

### ۲.۳۱ is_predicted
فقط برای سازگاری با نسخه‌های قبلی: user_logged ← false؛ بقیه ← true. مرجع معنا resolution_source است.
### ۲.۳۲ warnings
period_end_missing · period_end_missing_warning_cap_exceeded · period_end_missing_hard_cap_exceeded · low_history · high_cycle_variability · short_cycle_outlier_detected · long_cycle_outlier_detected · onboarding_data_used · default_values_used · prediction_based_output · predicted_period_overdue · cycle_unresolved_after_expected_period · insufficient_anchor_data · stale_onboarding_anchor · hormonal_contraception_active · postpartum_cycles_not_returned · perimenopause_age_range · long_absence_no_period · skipped_cycle_recorded · lh_bbt_conflict.
### ۲.۳۳ requires_user_input
true وقتی warnings شامل موردی است که بدون ورود یا تأیید کاربر رفع نمی‌شود: period_end_missing_warning_cap_exceeded · period_end_missing_hard_cap_exceeded · cycle_unresolved_after_expected_period · insufficient_anchor_data · contradictory_anchor_data · stale_onboarding_anchor.
### ۲.۳۴ days_to_ovulation
```
days_to_ovulation = difference_in_days(estimated_ovulation_date, target_date)
positive: days left | 0: ovulation day | negative: days since
null if ovulation cannot be computed, or days_late > 14, or fertility is suppressed (§2.42)
```

### ۲.۳۵ خروجی API
تاریخ‌ها میلادی ISO؛ معادل شمسی در کامنت. مثال: پریود ۱۰ تا ۱۴ تیر ۱۴۰۵ (۱ تا ۵ ژوئیه ۲۰۲۶)، سیکل ۲۹ روزه، target = ۲۷ تیر (۱۸ ژوئیه).
```
{
  "date": "2026-07-18",                       // 1405-04-27
  "cycle_day": 18,
  "main_phase": "luteal",
  "subphase": "early_luteal",
  "fertility_level": "low",
  "days_to_ovulation": -2,
  "days_to_period": 12,
  "days_late": 0,
  "anchors": {
    "current_period_start": "2026-07-01", "current_period_start_source": "user_logged",
    "current_period_end": "2026-07-05",   "current_period_end_source": "user_logged",
    "current_period_end_is_confirmed": true,
    "predicted_next_period_start": "2026-07-30",   // 1405-05-08
    "estimated_ovulation_date": "2026-07-16",      // 1405-04-25
    "ovulation_source": "calendar"
  },
  "metrics": { "effective_cycle_length": 29, "effective_period_length": 5,
               "cycle_variability": 3, "cycle_regularity": "regular" },
  "clinical_flags": { "frequency": "normal", "duration": "normal", "absent": false },
  "future_cycles": [ { "k": 2, "period_start": "2026-08-28", "ovulation": "2026-08-14" } ],
  "suggest_pregnancy_test": false,
  "confidence": "high", "confidence_reasons": [], "data_quality": "good",
  "resolution_source": "prediction", "is_predicted": true,
  "requires_user_input": false, "warnings": []
}
```

بررسی: تخمک‌گذاری روز ۱۶ سیکل (۳۰ − ۱۴). target روز ۱۸ = O+2 = اولین روز بازه assignable؛ بازه ۱۸ تا ۲۳ ژوئیه (۲۷ تیر تا ۱ مرداد) = ۶ روز؛ early_luteal = ۳ روز اول ← early_luteal. با وجود start و end واقعی، فاز از پیش‌بینی resolve شده است، پس resolution_source = prediction.
### ۲.۳۶ تا ۲.۳۸ مثال‌های قطعی (پریود ۵ روزه، بدون LH و BBT)

| روز | چرخه ۲۱ روزه | چرخه ۲۸ روزه | چرخه ۳۵ روزه |
|---|---|---|---|
| تخمک‌گذاری | روز ۸ | روز ۱۵ | روز ۲۲ |
| پنجره زیستی | روز ۳ تا ۸ | روز ۱۰ تا ۱۵ | روز ۱۷ تا ۲۲ |
| پنجره نمایشی | روز ۶ تا ۸ | روز ۱۰ تا ۱۵ | روز ۱۷ تا ۲۲ |
| menstrual | ۱–۵ | ۱–۵ | ۱–۵ |
| follicular | — (حذف) | ۶–۹ (early ۶، mid ۷، late ۸–۹) | ۶–۱۶ |
| fertile_rising | — (هم‌پوشانی با پریود) | ۱۰–۱۲ | ۱۷–۱۹ |
| high_fertility | ۶–۷ | ۱۳–۱۴ | ۲۰–۲۱ |
| ovulation_likely | ۸ | ۱۵ | ۲۲ |
| post_ovulation | ۹ | ۱۶ | ۲۳ |
| early / mid luteal | ۱۰–۱۲ / ۱۳–۱۵ | ۱۷–۱۹ / ۲۰–۲۲ | ۲۴–۲۶ / ۲۷–۲۹ |
| late_luteal | ۱۶–۱۸ | ۲۳–۲۵ | ۳۰–۳۲ |
| pms_possible | ۱۹–۲۱ | ۲۶–۲۸ | ۳۳–۳۵ |
| period_expected | از روز ۲۲ | از روز ۲۹ | از روز ۳۶ |

در چرخه‌های کوتاه ممکن است follicular یا fertile_rising نمایش داده نشوند؛ نمایش همه subphaseها تضمین‌شده نیست.
### ۲.۳۹ تأیید پیش‌بینی و recompute
- ۱. تاریخ پیش‌بینی‌شده هرگز خودکار به log واقعی تبدیل نمی‌شود.
- ۲. period_start واقعی همیشه سیکل تأییدشده جدید باز می‌کند.
- ۳. period_end واقعی همیشه بر assumed_period_end غلبه می‌کند.
- ۴. هر log تأییدشده جدید پیش‌بینی‌های بعدی را recompute می‌کند.
- ۵. مقادیر موثر فقط از history واقعی معتبر محاسبه می‌شوند.
- ۶. با period_start جدید، period_expected قبلی از آن تاریخ به بعد معتبر نیست.
- ۷. با period_end واقعی، همه خروجی‌های ساخته‌شده با assumed_end recompute می‌شوند.
- ۸. پریود رد شد: کاربر برای period_expected «پریود نداشتم» ثبت می‌کند؛ skipped_cycle ذخیره و از مقادیر موثر و نظم کنار گذاشته می‌شود؛ پیش‌بینی بعدی = پیش‌بینی قبلی + effective_cycle_length؛ warnings += skipped_cycle_recorded.
- ۹. ویرایش گذشته: هر تغییر در period_logs قدیمی، کل history، مقادیر موثر و کش تحلیل‌ها را از نو می‌سازد.
### ۲.۴۰ قواعد ممنوع
- ذخیره predicted_period_start در period_logs
- ذخیره assumed_period_end به‌عنوان period_end واقعی
- ovulation یا menstrual_possible به‌عنوان main_phase
- نمایش fertile روی روزهای menstrual
- محاسبه cycle_day به‌صورت 0-based
- current_period_start یا current_period_end بدون source؛ current_period_end بدون is_confirmed
- ورود داده predicted به مقادیر موثر
- محاسبه روی رشته تاریخ شمسی یا با ماه ثابت ۳۰ روزه؛ استفاده از UTC سرور به‌جای روز محلی
- نمایش پنجره باروری یا fertility_level وقتی hormonal_contraception = true
- استفاده از بازه ۲۱ تا ۴۵ به‌عنوان معیار «طبیعی» در UI
- ذخیره future_cycles
### ۲.۴۱ لنگر آنبوردینگ قدیمی
```
if no actual period_log AND anchor = last_period_start_from_onboarding
   AND difference_in_days(target_date, anchor) > effective_cycle_length + 14:
  n = floor(difference_in_days(target_date, anchor) / effective_cycle_length)
  current_period_start = anchor + n * effective_cycle_length
  current_period_start_source = predicted_reference
  confidence = low; warnings += stale_onboarding_anchor; requires_user_input = true
```

UI: بنر «آخرین پریودت کی شروع شد؟» با تقویم.
### ۲.۴۲ حالت‌های محدودکننده

| شرط | موتور | UI |
|---|---|---|
| hormonal_contraception = true | fertile و ovulation محاسبه نمی‌شوند؛ fertility_level = unknown؛ warnings += hormonal_contraception_active | پنجره باروری پنهان؛ «خون‌ریزی قطع مصرف» به‌جای پیش‌بینی پریود طبیعی |
| mode = pregnancy | موتور چرخه متوقف؛ منطق بارداری (§۳.۱) | حلقه ۴۰ هفته |
| mode = postpartum یا breastfeeding | پیش‌بینی فقط بعد از ۲ پریود واقعی پس از زایمان؛ warnings += postpartum_cycles_not_returned | پیام نامنظمی قابل انتظار |
| تست بارداری مثبت | prompt تغییر mode | «رفتن به حالت بارداری» |
| سن ≥ ۴۵ | confidence حداکثر medium؛ warnings += perimenopause_age_range | توضیح نامنظمی قابل انتظار |
| سن < ۱۸ | clinical_flags با بازه نوجوانان | — |

### ۲.۴۳ نگاشت خروجی موتور به صفحه امروز

| المان | قاعده |
|---|---|
| دکمه داخل حلقه «پریودم شروع شد» + «هنوز نه» | پریود باز نیست و target بین predicted_next − ۳ تا predicted_next + ۱۴ |
| دکمه «پریودم تموم شد» + «هنوز ادامه دارد» | end تأییدنشده و days_since ≥ effective_period_length − ۱ تا hard_cap |
| دکمه «ویرایش پریود» | بقیه روزها |
| بنر «پایان پریود ثبت نشده» | warnings شامل warning_cap_exceeded یا hard_cap_exceeded |
| بنر «پریود دیر کرده» | predicted_period_overdue (days_late ≥ ۸)؛ دکمه ثبت تست اگر suggest_pregnancy_test |
| بنر «آخرین پریودت کی بود؟» | requires_user_input با stale_onboarding_anchor یا cycle_unresolved_after_expected_period |
| حلقه | برچسب ≤ ۱۴ کاراکتر، عدد، واحد ≤ ۱۰ کاراکتر، زیرنویس ≤ ۱۸ کاراکتر؛ پیام‌های بلند فقط در بنر |
| کارت پیش‌بینی | پریود: predicted_next تا + effective_period_length − ۱ · PMS: predicted_next − ۳ تا − ۱ · پنجره باروری: display window · تخمک‌گذاری با source · طول سیکل + بج cycle_regularity |

حداکثر یک بنر همزمان به ترتیب جدول. «هنوز نه» و «هنوز ادامه دارد» فقط در حافظه محلی همان روز. هشدار ثابت: پیش‌بینی روش پیشگیری نیست.
## بخش ۳ — منطق ماژول‌های سلامت
### ۳.۱ بارداری و بعد از زایمان
- تاریخ زایمان از LMP: LMP + ۲۸۰ روز (نگله). اصلاح بر اساس طول سیکل اختیاری و پیش‌فرض خاموش است.
- از سونوگرافی: سن در روز سونو + روزهای گذشته؛ سونوی سه‌ماهه اول (CRL) دقیق‌ترین مبنا.

| سن بارداری هنگام سونو | جایگزینی LMP با سونو وقتی اختلاف بیش از |
|---|---|
| تا ۸ هفته و ۶ روز | ۵ روز |
| ۹+۰ تا ۱۳+۶ | ۷ روز |
| ۱۴+۰ تا ۱۵+۶ | ۷ روز |
| ۱۶+۰ تا ۲۱+۶ | ۱۰ روز |
| ۲۲+۰ تا ۲۷+۶ | ۱۴ روز |
| از ۲۸+۰ | ۲۱ روز |

- سه‌ماهه‌ها: اول تا ۱۳+۶ · دوم ۱۴+۰ تا ۲۷+۶ · سوم از ۲۸+۰ (ACOG CO 700).

| BMI قبل از بارداری (WHO) | افزایش کل توصیه‌شده (IOM 2009، تک‌قلو) |
|---|---|
| کمتر از ۱۸٫۵ | ۱۲٫۵ تا ۱۸ کیلو |
| ۱۸٫۵ تا ۲۴٫۹ | ۱۱٫۵ تا ۱۶ کیلو |
| ۲۵ تا ۲۹٫۹ | ۷ تا ۱۱٫۵ کیلو |
| ۳۰ و بیشتر | ۵ تا ۹ کیلو |

- حرکات جنین (از ۲۸ هفته): ۱۰ حرکت در حداکثر ۲ ساعت؛ کمتر یا کاهش محسوس الگو ← «همین حالا با پزشک یا ماما تماس بگیر».
- انقباض: مدت = شروع تا پایان؛ فاصله = شروع تا شروع بعدی. الگوی ۵-۱-۱ (هر ۵ دقیقه، هر کدام حدود ۱ دقیقه، به مدت ۱ ساعت)، پارگی کیسه آب یا خون‌ریزی ← «با زایشگاه تماس بگیر».
- فشار خون در بارداری (ACOG PB 222): ≥ ۱۴۰ یا ≥ ۹۰ ← «با پزشک تماس بگیر»؛ ≥ ۱۶۰ یا ≥ ۱۱۰ ← «فوری». جایگزین جدول §۳.۲ در حالت بارداری.
- قند در دیابت بارداری (ADA بخش ۱۵): ناشتا < ۹۵ · ۱ ساعت بعد غذا < ۱۴۰ · ۲ ساعت بعد غذا < ۱۲۰ mg/dL.
- خون‌ریزی در بارداری: هر ثبت ← «با پزشک در میان بگذار»؛ زیاد همراه درد ← «فوری».
- بعد از زایمان: پر شدن یک نوار در یک ساعت یا کمتر، تب ≥ ۳۸°C، درد شدید یا تنگی نفس ← فوری. ویزیت: تماس در ۳ هفته اول، ویزیت جامع تا ۱۲ هفته (ACOG CO 736)؛ یادآور پیش‌فرض ۶ هفته.
- EPDS کامل ۱۰ سؤالی (Cox 1987)، هر ۲ هفته: ≥ ۱۰ احتمال افسردگی ← «با پزشک صحبت کن»؛ ≥ ۱۳ احتمال بالا. هر پاسخ غیر صفر به سؤال ۱۰ ← صفحه فوری با ۱۲۳ و ۱۱۵، مستقل از نمره کل.
### ۳.۲ علائم حیاتی
هر ثبت با زمینه (زمان، حالت، ناشتا/بعد غذا) ذخیره و رده‌بندی می‌شود؛ گزارش‌ها بر اساس میانگین حداقل ۲ ثبت در ۲ روز.

| فشار خون (ACC/AHA 2017؛ همان در 2025) | سیستولیک |  | دیاستولیک |
|---|---|---|---|
| طبیعی | کمتر از ۱۲۰ | و | کمتر از ۸۰ |
| بالا | ۱۲۰ تا ۱۲۹ | و | کمتر از ۸۰ |
| مرحله ۱ | ۱۳۰ تا ۱۳۹ | یا | ۸۰ تا ۸۹ |
| مرحله ۲ | ۱۴۰ و بیشتر | یا | ۹۰ و بیشتر |
| بحران | بیشتر از ۱۸۰ | و/یا | بیشتر از ۱۲۰ |

- رده بالاتر برنده است. بحران ← «۵ دقیقه صبر کن و دوباره اندازه بگیر؛ اگر هنوز بالاست یا درد قفسه سینه، تنگی نفس، ضعف یا اختلال گفتار داری، ۱۱۵».
- ضربان در حال استراحت (AHA): ۶۰ تا ۱۰۰؛ کمتر ← «پایین‌تر از معمول» (در ورزشکاران می‌تواند طبیعی باشد)؛ بیشتر ← «بالاتر از معمول». فقط ثبت‌های «استراحت».

| قند (ADA 2025، بخش ۲) | طبیعی | پیش‌دیابت | دیابت |
|---|---|---|---|
| ناشتا (≥ ۸ ساعت) | کمتر از ۱۰۰ | ۱۰۰ تا ۱۲۵ | ۱۲۶ و بیشتر |
| ۲ ساعت OGTT ۷۵ گرمی | کمتر از ۱۴۰ | ۱۴۰ تا ۱۹۹ | ۲۰۰ و بیشتر |
| تصادفی | — | — | ۲۰۰ و بیشتر + علائم |
| HbA1c | کمتر از ۵٫۷٪ | ۵٫۷ تا ۶٫۴٪ | ۶٫۵٪ و بیشتر |

- مثال: ناشتای ۱۰۰ در بازه پیش‌دیابت است؛ پیام «بالاتر از بازه طبیعی ناشتا؛ یک عدد تکی تشخیص نیست؛ تکرار یا آزمایش را با پزشک هماهنگ کن». همان ۱۰۰ بعد از غذا در بازه معمول است.
- بعد از غذا با گلوکومتر (غیردیابتی): ADA آستانه تشخیصی ندارد؛ کمتر از ۱۴۰ «در بازه معمول»، بیشتر «با پزشک در میان بگذار». دیابتی: هدف پیش‌فرض ADA قبل غذا ۸۰ تا ۱۳۰ و اوج بعد غذا کمتر از ۱۸۰، قابل جایگزینی با هدف پزشک.
- افت قند: کمتر از ۷۰ سطح ۱؛ کمتر از ۵۴ سطح ۲ (فوری: «۱۵ گرم قند سریع‌الجذب و تکرار بعد از ۱۵ دقیقه»). mmol/L = mg/dL ÷ ۱۸.
- BMI (WHO): کمتر از ۱۸٫۵ کم‌وزن · ۱۸٫۵ تا ۲۴٫۹ طبیعی · ۲۵ تا ۲۹٫۹ اضافه‌وزن · ۳۰ و بیشتر چاقی. در بارداری فقط BMI قبل از بارداری.
### ۳.۳ تحلیل آزمایش با هوش مصنوعی
جریان: رضایت صریح ← بارگذاری عکس یا PDF ← OCR (نام شاخص، عدد، واحد، محدوده مرجع) ← تأیید کاربر ← رده‌بندی قاعده‌محور ← توضیح ساده با LLM و سؤال برای پزشک.
- اولویت محدوده مرجع: برگه همان آزمایشگاه ← جدول پیش‌فرض با برچسب «محدوده عمومی» ← بدون رده.
- OCR: مقدار با اطمینان کمتر از ۰٫۹ کهربایی نمایش داده می‌شود و تا تأیید کاربر وارد تحلیل نمی‌شود.
- LLM: ورودی = مقادیر تأییدشده + رده‌ها + زمینه (سن، جنس، حالت، داروها)؛ خروجی JSON (خلاصه، توضیح شاخص‌های خارج از بازه، سؤال برای پزشک). رده را تغییر نمی‌دهد، نام بیماری را به‌عنوان نتیجه نمی‌گوید، دوز پیشنهاد نمی‌کند؛ فیلتر کلمات تشخیصی روی خروجی.

| شاخص | آستانه پیش‌فرض | منبع |
|---|---|---|
| هموگلوبین، زن غیرباردار | کمتر از ۱۲ g/dL | WHO 2024 |
| هموگلوبین، بارداری | کمتر از ۱۱ g/dL (سه‌ماهه دوم کمتر از ۱۰٫۵) | WHO 2024 |
| فریتین، سالم / با التهاب | کمتر از ۱۵ / کمتر از ۷۰ ng/mL | WHO 2020 |
| ویتامین D (25-OH) | کمتر از ۱۲ کمبود؛ ۲۰ و بیشتر کافی | IOM 2011 |
| TSH در بارداری بدون محدوده سه‌ماهه | حد بالا ۴٫۰ mIU/L | ATA 2017 |
| قند ناشتا و HbA1c | جدول §۳.۲ | ADA 2025 |

- Endocrine Society 2024 برای ویتامین D آستانه ثابت پیشنهاد نمی‌کند (نسخه 2011 کمتر از ۲۰ را کمبود می‌دانست)؛ این نکته در توضیح نمایش داده شود.
- فایل و نتیجه جدا ذخیره و قابل حذف‌اند. جمله «نام و شماره ارسال نمی‌شود» در متن رضایت فقط در صورت ناشناس‌سازی واقعی در بک‌اند مجاز است.
### ۳.۴ چکاپ‌ها و غربالگری

| غربالگری | سن | فاصله | منبع |
|---|---|---|---|
| پاپ‌اسمیر (سیتولوژی) | ۲۱ تا ۲۹ | هر ۳ سال | USPSTF 2018 |
| HPV اولیه یا هم‌آزمایی | ۳۰ تا ۶۵ | هر ۵ سال (یا سیتولوژی هر ۳ سال) | USPSTF 2018 |
| ماموگرافی | ۴۰ تا ۷۴ | هر ۲ سال | USPSTF 2024 |
| فشار خون | ۱۸ تا ۳۹ کم‌خطر / ۴۰ و بیشتر یا پرخطر | هر ۳ تا ۵ سال / سالانه | USPSTF 2021 |
| قند | ۳۵ تا ۷۰ با اضافه‌وزن | هر ۳ سال | USPSTF 2021 |
| دندان‌پزشکی | همه | طبق نظر دندان‌پزشک (معمولاً ۶ تا ۱۲ ماه) | American Dental Association |

- موعد بعدی = تاریخ آخرین انجام + فاصله؛ قابل تغییر توسط کاربر یا پزشک. «نیاز به پیگیری» فاصله پیش‌فرض را غیرفعال می‌کند.
- «آگاهی از سینه» به‌جای خودآزمایی ماهانه (ACOG PB 179). در بارداری غربالگری‌های غیرفوری با برچسب «پس از بارداری».
### ۳.۵ کودک
- رشد: WHO Child Growth Standards (۰ تا ۵ سال، LMS)؛ کمتر از صدک ۳، بیشتر از صدک ۹۷، یا عبور از ۲ خط صدک اصلی بین دو اندازه‌گیری ← «با پزشک کودک در میان بگذار».
- مراحل رشد: CDC 2022؛ ۲، ۴، ۶، ۹، ۱۲، ۱۵، ۱۸، ۲۴ و ۳۰ ماه و ۳، ۴ و ۵ سال؛ هر مرحله مهارتی که حدود ۷۵٪ کودکان دارند.
- واکسن: برنامه کشوری وزارت بهداشت به‌صورت پیکربندی نسخه‌دار؛ جدول طرح نمونه است؛ کارت واکسن مرجع نهایی.
- شیردهی، خواب و پوشک: فقط ثبت و خلاصه؛ بدون رده‌بندی.
## بخش ۴ — تحلیل، رضایت و مدل درآمد
### ۴.۱ تب تحلیل

| حالت | صفحه‌ها | تحلیل‌ها |
|---|---|---|
| سیکل | An_Hub، An_Cycle، An_Period، An_Symptoms، An_Correlations، An_Body، An_Labs، An_Monthly | طول و نظم سیکل، شدت پریود روز به روز، الگوی علائم، ارتباط‌ها، وزن/خواب/فعالیت، روند آزمایش، گزارش ماهانه |
| اقدام به بارداری | An_Hub_TTC، An_Fertility | سیکل‌های اقدام، BBT با خط پوشش و تأیید، LH، زمان‌بندی رابطه، ترشحات، طول لوتئال. راهنما: زیر ۳۵ سال بعد از ۱۲ ماه و ۳۵ به بالا بعد از ۶ ماه اقدام بدون نتیجه با پزشک مشورت کن |
| بارداری | An_Hub_Preg، An_PregWeight | وزن روی باند IOM، فشار با آستانه بارداری، قند با اهداف دیابت بارداری، حرکات جنین، علائم هر سه‌ماهه |
| بعد از زایمان | An_Hub_Post | روند EPDS، کاهش خون‌ریزی، شیردهی، خواب مادر و نوزاد، رشد کودک، وزن |

| تحلیل | روش | حداقل داده |
|---|---|---|
| طول، نظم، پرچم‌ها | از موتور (§۲.۹، §۲.۲۷) | ۲ سیکل؛ نظم ۳ سیکل در ۱۲ ماه |
| شدت پریود | میانگین شدت هر روز پریود | ۲ پریود |
| الگوی علائم | فراوانی نسبی علامت در هر روز سیکل | علامت در ≥ ۳ سیکل |
| حال در فازها | درصد روزهای هر حال در هر فاز | ≥ ۱۰ روز در هر فاز |
| ارتباط‌ها | نسبت دو گروه روز؛ قوی: ≥ ۳۰ روز و اختلاف ≥ ۲ برابر | ۳۰ روز داده جفت‌شده |
| وزن | میانگین متحرک ۷ روزه؛ تغییر ۷ و ۳۰ روزه | ۳ ثبت در ۱۴ روز |
| فشار و قند | میانگین، کمینه–بیشینه، توزیع رده‌ها | ۲ ثبت در ۲ روز |
| روند آزمایش | همان شاخص در آزمایش‌های پشت‌سرهم | ۲ آزمایش |
| BBT | سه بالای شش؛ نتیجه به موتور برمی‌گردد | ≥ ۱۰ دما در سیکل |
| وزن بارداری | مقایسه با باند IOM | BMI قبل بارداری + ۲ ثبت |
| EPDS | نمره و روند | ۱ نمره؛ روند با ۲ |
| گزارش ماهانه | شاخص‌های ماه + تغییر نسبت به ماه قبل | ۱۰ روز ثبت |

- متن همه ارتباط‌ها: «هم‌زمانی در داده‌های تو، نه علت». مهم‌ترین یافته بالای هر هاب = بیشترین قوت × تازگی؛ در نبود داده: «برای دیدن الگوها N روز دیگر ثبت کن».
- محاسبه سمت سرور در job شبانه یا هنگام باز شدن تب اگر داده تغییر کرده؛ ویرایش گذشته کش را باطل می‌کند.
### ۴.۲ رضایت، حریم خصوصی و ایمنی

| رضایت | زمان دریافت | دامنه |
|---|---|---|
| قوانین + پردازش داده سلامت | ثبت‌نام | اجباری |
| تحلیل آزمایش با هوش مصنوعی | اولین استفاده (یا هر بار) | تصویر آزمایش + سن، جنس، حالت، داروها |
| دستیار سلامت از پروفایل | ورود به هر بخش دستیار؛ هر مورد با سوییچ | موارد روشن‌شده |
| اشتراک با پزشک | هر رزرو یا ارجاع | فقط همان پزشک و ویزیت |
| ریتمی همراه و خانواده | دعوت هر عضو و هر تغییر دسترسی | به تفکیک دامنه و سطح (§۱.۷) |
| لینک گزارش PDF | ساخت لینک | ۷ روز، قابل لغو |

- مدل داده: consent(user_id, type, version, granted_at, revoked_at, scope_json).
- اعلان‌های خنثی روی صفحه قفل («یادآور ریتمی»)، قفل اپ، مات شدن پیش‌نمایش در برنامه‌های اخیر، حذف کامل داده از دستگاه و سرور با تأیید دوباره.
- مسیرهای فوری ثابت و بدون هوش مصنوعی: فشار بحرانی، قند کمتر از ۵۴، کاهش حرکات جنین، الگوی انقباض، خون‌ریزی شدید بعد از زایمان، سؤال ۱۰ EPDS. شماره‌ها: ۱۱۵ و ۱۲۳.
### ۴.۳ پریمیوم و مدل درآمد
هسته ثبت، پیش‌بینی پایه و ایمنی رایگان است؛ درآمد از عمق (تحلیل، هوش مصنوعی، گزارش) و بازارگاه (پزشک، مدرس). قیمت‌ها فرضیه‌اند و با آزمون A/B تأیید می‌شوند.

| بخش | رایگان | ریتمی پلاس |
|---|---|---|
| سیکل و TTC | ثبت کامل، پیش‌بینی پریود و پنجره باروری، تقویم، یادآورها | منبع و اطمینان پیش‌بینی، BBT با تأیید، پیش‌بینی با LH |
| بارداری و بعد از زایمان | هفته‌به‌هفته، حرکات، انقباض، EPDS، مسیرهای فوری | محتوای عمیق، روند وزن با IOM |
| ثبت (+) | همه دسته‌ها | دسته‌های سفارشی نامحدود |
| تحلیل | طول سیکل، تاریخچه، گزارش ماهانه ساده | الگوی علائم، ارتباط‌ها، حال در فازها، روند آزمایش، بازه ۱ ساله، کارت‌های پلاس هر حالت |
| علائم حیاتی | ثبت و رده‌بندی | گزارش روند، توزیع، صبح/شب |
| پرونده سلامت | پرونده | PDF و لینک اشتراک |
| تحلیل آزمایش (AI) | ۱ در ماه | ۱۰ در ماه |
| دستیار سلامت (AI) | ۵ پیام در ماه | نامحدود با سقف منصفانه |
| تله‌مدیسین | قیمت کامل | تخفیف هر ویزیت |
| لیست کارها، ریتمی همراه و خانواده، آموزش رایگان مدرس‌ها | کامل | امکانات پلاس مرتبط با داده مشترک برای اعضا |

- قیمت پیشنهادی: ۱ ماهه ۹۹ هزار · ۳ ماهه ۲۳۷ هزار (ماهانه ۷۹، ۲۰٪ صرفه‌جویی، محبوب‌ترین) · ۶ ماهه ۳۹۰ هزار (ماهانه ۶۵، ۳۴٪) تومان · ۷ روز آزمایشی فقط اولین اشتراک · تست پلن ۱۲ ماهه.
- سایر درآمدها: کمیسیون تله‌مدیسین · اشتراک یا کمیسیون مدرس · بسته اعتباری هوش مصنوعی · B2B2C با بیمه‌ها و کارفرماها با کد فعال‌سازی.
- جای paywall: پایان ثبت‌نام (نرم، قابل رد) · کارت‌های پلاس تحلیل · لحظه ارزش مثل «ساخت PDF برای پزشک» · اتمام سهمیه هوش مصنوعی. هرگز روی مسیرهای فوری، یادآور دارو یا EPDS.
- شاخص‌ها: شروع آزمایشی، تبدیل به پولی، ماندگاری ماه ۱ و ۳، سهم پلن‌ها، هزینه هوش مصنوعی به ازای هر کاربر پولی.
## بخش ۵ — پنل ادمین (وب)
پنل وب ۱۴۴۰ پیکسلی با منوی کناری: داشبورد، کاربران، پیام‌ها، موتور پیام و شبیه‌ساز، چالش‌ها، KPI و تحلیل، OKR، ایمنی و محتوای بالینی، آستانه‌ها و پیکربندی، نقش‌ها و ممیزی؛ و ماژول‌های اشتراک، پزشکان و مدرسین. صفحه‌ها در کانواس با پیشوند Admin_.
### ۵.۱ داشبورد
- کارت‌ها: DAU، MAU، DAU/MAU، ثبت‌نام امروز، مشترک فعال پلاس (با نرخ تبدیل آزمایشی)، درآمد ماهانه تکراری؛ هر کدام با تغییر نسبت به بازه قبل.
- نمودار DAU و کاربران ثبت‌کننده روزانه؛ کاربران بر اساس mode؛ سلامت موتور (§۵.۶)؛ هشدارهای فعال؛ خلاصه OKR.
- هشدار خودکار وقتی شاخص از آستانه تعریف‌شده عبور کند: نرخ اصلاح OCR بیش از ۱۲٪، افت CTR پیام بیش از ۳۰٪ نسبت به میانگین ۷ روزه، افزایش کاربران requires_user_input بیش از ۷ روز، خطای میانگین پیش‌بینی بیش از ۲٫۵ روز.
### ۵.۲ رصد کاربران
- لیست: شناسه مستعار (RT-xxxx)، موبایل ماسک‌شده، mode، پلن، آخرین فعالیت، روزهای ثبت در ۳۰ روز، data_quality، هشدار اصلی موتور، ریسک ریزش. فیلترها قابل ذخیره به‌عنوان سگمنت (برای پیام‌ها و تحلیل).
- ریسک ریزش (MVP): بالا اگر روزهای ثبت در ۱۴ روز اخیر ≤ ۵۰٪ ۱۴ روز قبل یا ۷ روز بدون فعالیت؛ متوسط اگر ≤ ۷۵٪؛ در غیر این صورت کم.
- جزئیات کاربر: حساب، خروجی موتور امروز، تاریخچه طول سیکل، پیام‌ها و چالش‌های اخیر با قاعده‌ای که باعث نمایش شد، اشتراک، رضایت‌ها، پشتیبانی.
- داده سلامت (علائم، آزمایش، یادداشت، گفت‌وگوی دستیار) به‌صورت پیش‌فرض مخفی است؛ فقط نقش «پشتیبانی بالینی» با ثبت دلیل و شماره تیکت، ۳۰ دقیقه دسترسی می‌گیرد؛ در لاگ ممیزی ثبت می‌شود. خروجی داده فقط ناشناس.
### ۵.۳ موتور پیام: مدل داده
```
message { id, version, name, type, slot, priority, status,
          content { title[variant], body, cta_text, cta_deeplink, variables[] },
          rule { all_of[condition], any_of[group], exclude[condition] },
          frequency { max_per_cycle, max_per_week, cooldown_days },
          schedule { start_at, end_at, local_hours },
          audience_percent, ab_variants[{key, weight}], goal_event,
          clinical_review { required, approved_by, approved_version } }
condition { field, op, value }   // op: = != in not_in >= <= between contains
message_event { user_id, message_id, version, variant, slot, event, at }
                // event: eligible | shown | clicked | dismissed | goal
```

| type | اولویت مجاز | قواعد |
|---|---|---|
| safety | ۱۰۰۰ | همیشه اول؛ بدون سقف تکرار، A/B و سهم مخاطب؛ فقط مشاور پزشکی ویرایش و منتشر می‌کند |
| engine | ۸۰۰ تا ۹۹۹ | بنرهای وضعیت خانه از warnings موتور (§۲.۴۳)؛ حداکثر یکی |
| clinical / content | ۱۰۰ تا ۷۹۹ | نیاز به تأیید مشاور پزشکی؛ هر تغییر متن تأیید را باطل می‌کند |
| product / marketing | ۱ تا ۹۹ | سقف روزانه و هفتگی؛ هرگز در ۲۴ ساعت بعد از پیام ایمنی |

سطح پیام برای کاربر (level): مستقل از اولویت؛ ظاهر و رفتار پیام را تعیین می‌کند و در صفحه «هشدارها و پیام‌ها» و کارت‌ها یکسان نمایش داده می‌شود.

| level | معنا | قواعد اجباری | typeهای مجاز |
|---|---|---|---|
| اطلاع (info) | اتفاق عادی؛ نیاز به کاری ندارد | قابل بستن؛ CTA اختیاری | همه به‌جز safety |
| پیشنهاد (suggestion) | یک کار کوچک که داده یا تحلیل را کامل‌تر می‌کند | یک CTA؛ قابل بستن | engine، content، product |
| ارزش پیگیری (follow_up) | الگوی تکرارشونده؛ بهتر است با متخصص در میان گذاشته شود | فیلدهای «چی دیدیم» و «چقدر مطمئنیم» + گزینه «افزودن به یادداشت ویزیت»؛ تأیید مشاور پزشکی | engine، clinical |
| پیگیری زودتر (urgent) | شرایطی که نباید نادیده گرفته شود | دکمه تماس (۱۱۵ یا پزشک) اجباری؛ غیرقابل بستن تا دیده شدن؛ تأیید مشاور پزشکی | safety، clinical |

فیلدهای مجاز در شرط‌ها: خروجی موتور (mode، main_phase، subphase، cycle_day، days_to_period، days_late، confidence، data_quality، warnings، requires_user_input، cycle_regularity، suggest_pregnancy_test، hormonal_contraception)، هفته بارداری و سه‌ماهه، هفته بعد از زایمان، آخرین رده علائم حیاتی، symptom_history (تعداد سیکل‌هایی که علامت در همان فاز ثبت شده)، plan، days_since_signup، days_since_last_log، logs_30d، سگمنت، سن، push_opt_in.
### ۵.۴ موتور پیام: کی چه پیامی نمایش داده می‌شود

| گام | قاعده |
|---|---|
| ۱. رویداد | باز شدن اپ، ثبت داده، زمان‌بندی پوش، تغییر mode |
| ۲. وضعیت | خروجی موتور چرخه و ماژول‌ها برای target_date (روز محلی) |
| ۳. کاندیدها | پیام‌های status = active همان slot، داخل schedule و local_hours |
| ۴. ارزیابی | all_of همه برقرار ∧ حداقل یکی از any_of (اگر تعریف شده) ∧ هیچ exclude برقرار نیست ∧ hash(user_id, message_id) mod 100 < audience_percent |
| ۵. سقف | حذف اگر max_per_cycle، max_per_week یا cooldown_days (از آخرین shown یا dismissed) پر شده باشد |
| ۶. انتخاب | safety اول؛ سپس priority بیشتر؛ در تساوی، پیامی که دیرتر نمایش داده شده؛ سپس شناسه کوچک‌تر |
| ۷. نمایش | variant = hash(user_id, message_id) روی وزن‌ها (ثابت برای هر کاربر)؛ لاگ eligible و shown |

| slot | ظرفیت | نوع مجاز | سقف پیش‌فرض |
|---|---|---|---|
| مودال فوری | ۱ | safety | بدون سقف |
| بنر وضعیت خانه | ۱ | engine | تا رفع وضعیت |
| نکته هوشمند | ۱ | clinical | ۱ در سیکل برای هر پیام |
| توصیه امروز | ۳ | content | روزانه، بدون تکرار ۷ روز |
| چالش امروز | ۱ | product | تا پایان چالش |
| پوش | ۱ در روز، ۳ در هفته | همه | ساعت سکوت ۲۳ تا ۸ |
| مودال درون‌اپ | ۱ در هفته | marketing | — |

### ۵.۴.۱ صفحه «هشدارها و پیام‌ها» از همان موتور پیام
هشدارهای بارداری (و هر حالت دیگر) موتور جداگانه ندارند: هر هشدار یک message با slot = alerts_inbox است و از همان خط لوله §۵.۴ عبور می‌کند. تفاوت این slot با بقیه این است که پیام‌ها فهرست می‌شوند، نه اینکه فقط یک برنده نمایش داده شود.

| type (کلاس اولویت) | levelهای مجاز | نمونه در بارداری |
|---|---|---|
| safety (۱۰۰۰) | پیگیری زودتر | فشار ≥ ۱۶۰/۱۱۰ · کاهش حرکات جنین · الگوی انقباض ۵-۱-۱ · خون‌ریزی |
| engine (۸۰۰ تا ۹۹۹) | پیشنهاد، ارزش پیگیری | مبنای سن بارداری ناقص · تاریخ زایمان نیاز به بازبینی بعد از سونو (§۳.۱) |
| clinical (۱۰۰ تا ۷۹۹) | ارزش پیگیری، پیگیری زودتر، اطلاع | استفراغ مکرر · فشار ≥ ۱۴۰/۹۰ · افزایش وزن خارج از باند IOM · قند بالاتر از اهداف دیابت بارداری |
| content / product (۱ تا ۹۹) | اطلاع، پیشنهاد | ورود به هفته جدید · وزن این هفته ثبت نشده · ویزیت بعدی |
| marketing | — | در این slot مجاز نیست |

قواعد slot alerts_inbox:
- ترتیب: پیگیری زودتر ← ارزش پیگیری ← پیشنهاد ← اطلاع؛ در هر سطح، جدیدتر بالاتر.
- شواهد: برای level ≥ ارزش پیگیری، فیلدهای evidence.what («چی دیدیم»، مثلاً «استفراغ در ۴ ثبت اخیر، دو بار با شدت شدید») و evidence.basis («این الگو از ثبت‌های خودته، نه تشخیص») از داده همان کاربر پر می‌شوند و بدون آن‌ها پیام ساخته نمی‌شود.
- چرخه عمر: active ← acknowledged («دیدم، ممنون») یا added_to_visit_note («افزودن به یادداشت ویزیت») یا resolved (شرط دیگر برقرار نیست) ← archived بعد از ۳۰ روز. پیام پیگیری زودتر تا acknowledged نشود بسته نمی‌شود.
- ادغام: اگر همان message_id دوباره برقرار شود، پیام قبلی به‌روز می‌شود (شمارش و تاریخ)، پیام تکراری ساخته نمی‌شود.
- هم‌زمانی با slotهای دیگر: پیام safety هم در مودال فوری و هم در صندوق ثبت می‌شود؛ پیام engine هم در بنر خانه و هم در صندوق. بقیه فقط در صندوق.

| هشدار بارداری (صفحه PregV3_Alerts) | type | level | قاعده نمونه |
|---|---|---|---|
| استفراغ ۴ روز پشت سر هم | clinical | ارزش پیگیری | vomiting در ≥ ۴ روز از ۷ روز اخیر، یا شدت «شدید» ≥ ۲ بار؛ اگر «مایعات را نگه نمی‌دارم» ثبت شود ← safety / پیگیری زودتر (ACOG PB 189) |
| وزن این هفته ثبت نشده | product | پیشنهاد | mode = pregnancy ∧ days_since_weight ≥ ۷ |
| وارد هفته ۸ شدی | content | اطلاع | روز اول هر هفته بارداری |
| ویزیت بعدی | content | اطلاع | visit.date − today ≤ ۳ روز |
| کاهش حرکات جنین | safety | پیگیری زودتر | از ۲۸ هفته، کمتر از ۱۰ حرکت در ۲ ساعت (§۳.۱) |
| فشار بالا در بارداری | clinical / safety | ارزش پیگیری / پیگیری زودتر | ≥ ۱۴۰/۹۰ / ≥ ۱۶۰/۱۱۰ (§۳.۱) |

- قواعد ثابت: وقتی requires_user_input = true فقط safety و engine نمایش داده می‌شوند. پیام clinical با confidence = low نمایش داده نمی‌شود مگر شرطش صریحاً اجازه دهد. پیام‌هایی که به پنجره باروری اشاره دارند وقتی hormonal_contraception = true حذف می‌شوند.
- متن فوری ثابت: متن مسیرهای فوری (§۴.۲) در اپ bundle است و از سرور قابل غیرفعال شدن نیست؛ پنل فقط نسخه متن را با تأیید مشاور پزشکی عوض می‌کند.
- شبیه‌ساز: با انتخاب کاربر و تاریخ، خروجی موتور و برنده هر slot به‌همراه دلیل حذف بقیه نمایش داده می‌شود. بررسی تداخل در ویرایشگر: پیام‌های هم‌slot با شرط هم‌پوشان و اولویت برنده.
### ۵.۵ چالش‌ها
```
challenge { id, title, duration_days, start_rule: on_join | on_phase_start,
            eligibility: rule, contraindications: rule, daily_tasks[], reminder_local_time,
            completion_rule { min_days }, reward, status }
participation { user_id, challenge_id, started_at, days_done[], status }
```

- پیشنهاد چالش از طریق slot «چالش امروز» با همان موتور پیام انجام می‌شود. contraindications همیشه بر eligibility غلبه دارد (مثلاً ورزش شدید در بارداری یا کمتر از ۶ هفته بعد از زایمان).
- تکمیل = days_done ≥ min_days در بازه duration؛ آمار: شرکت‌کننده، نرخ تکمیل، ریزش روز به روز.
### ۵.۶ KPI و مانیتورها (تعاریف)

| شاخص | تعریف |
|---|---|
| DAU / MAU | کاربران یکتا با حداقل یک session در روز / ۳۰ روز |
| کاربر ثبت‌کننده | حداقل یک ثبت داده در روز |
| قیف فعال‌سازی | نصب ← OTP ← آنبوردینگ کامل ← اولین ثبت ← ثبت در روز ۷ ← ثبت در روز ۳۰ (درصد از نصب) |
| نگهداشت Dn | درصد کوهورت ثبت‌نام که در روز n (±۱) حداقل یک ثبت داشته‌اند |
| خطای پیش‌بینی (MAE) | میانگین \|شروع واقعی − predicted_next_period_start\| برای سیکل‌هایی که پیش‌بینی از ≥ ۲ سیکل واقعی ساخته شده |
| دقت ±۲ روز | درصد همان سیکل‌ها با خطای ≤ ۲ روز |
| توزیع confidence و data_quality | از خروجی روزانه موتور برای کاربران فعال |
| requires_user_input حل‌شده | درصد مواردی که ظرف ۷ روز با ورودی کاربر رفع شده‌اند |
| تبدیل آزمایشی | مشترکان پولی / شروع‌کنندگان آزمایشی ۷ روزه در همان کوهورت |
| MRR | مجموع ارزش ماهانه اشتراک‌های فعال (پلن ۳ و ۶ ماهه تقسیم بر مدت) |
| هزینه AI | هزینه API تحلیل آزمایش و دستیار / درآمد پلاس |
| نرخ اصلاح OCR | درصد مقادیری که کاربر در صفحه تأیید ویرایش کرده |
| CTR پیام | clicked / shown برای هر نسخه پیام |
| خانواده | نرخ پذیرش دعوت · میانگین اعضای هر حلقه · نرخ تأیید پیشنهادها · نرخ ثبت دوز بعد از یادآوری ملایم |

### ۵.۷ OKR
- هر Objective چند KR دارد و هر KR به یک شاخص §۵.۶ با مبدأ، هدف و بازه وصل است؛ پیشرفت = (فعلی − مبدأ) / (هدف − مبدأ)، محدود به ۰ تا ۱۰۰٪.
- وضعیت: «در مسیر» اگر پیشرفت ≥ درصد زمان سپری‌شده فصل − ۱۰؛ «در خطر» اگر کمتر؛ «انجام شد» در ۱۰۰٪.
- نمونه پاییز ۱۴۰۵: O1 عادت روزانه (نگهداشت D30 از ۲۹ به ۳۵٪، ثبت‌کنندگان ≥ ۱۵ روز در ماه به ۴۵٪) · O2 پیش‌بینی قابل اعتماد (دقت ±۲ روز به ۷۵٪، confidence پایین زیر ۱۲٪) · O3 درآمد پایدار (۱۵ هزار مشترک، تبدیل آزمایشی ۴۰٪، هزینه AI زیر ۱۵٪ درآمد).
### ۵.۸ ایمنی، آستانه‌ها، نقش‌ها و ممیزی
- ایمنی: فقط آمار تجمیعی مسیرهای فوری (بدون شناسه)، صف بازبینی محتوای بالینی، و پایش سلامت مسیرهای فوری (زمان نمایش، تست خودکار شبانه همه سناریوها).
- رجیستری آستانه‌ها: threshold { key, value, unit, source, version, effective_from, approvals[] }. موتور مقداری را می‌خواند که در target_date معتبر است؛ تغییر آستانه بالینی با ۲ تأیید مشاور پزشکی.

| نقش | دسترسی |
|---|---|
| مدیر محصول | داشبورد، KPI، OKR، کاربران ماسک‌شده، پیام‌های غیربالینی، چالش‌ها |
| مشاور پزشکی | تأیید پیام بالینی و آستانه، انتشار پیام ایمنی، مشاهده داشبورد |
| پشتیبانی | کاربران ماسک‌شده، اشتراک، تیکت‌ها |
| پشتیبانی بالینی | دسترسی موقت ۳۰ دقیقه‌ای به داده سلامت با دلیل و تیکت |
| تحلیل‌گر | KPI، خروجی ناشناس |
| بازاریابی | پیام‌های product/marketing، چالش‌ها، سگمنت‌ها |
| مهندسی | پیکربندی فنی، مشاهده آستانه‌ها |

- لاگ ممیزی: هر ایجاد، ویرایش، تأیید، انتشار، خروجی داده و دسترسی به داده سلامت با زمان، کاربر پنل و جزئیات؛ غیرقابل حذف.
## بخش ۶ — تست‌ها و منابع
### ۶.۱ تست‌های واحد لازم
- چرخه ۲۱، ۲۸ و ۳۵ روزه طبق جدول §۲.۳۶
- پریود بدون end تا روز ۱۳ و بعد از hard cap
- days_late = ۷، ۸ و ۱۵
- پنجره fertile خالی؛ بازه luteal assignable خیلی کوتاه
- نبود history با onboarding؛ نبود onboarding با default
- median دو مقدار با .5
- مرز ماه‌های شمسی ۳۱ و ۳۰ روزه و اسفند کبیسه؛ مثال §۲.۳۵
- روز O+1 خارج از پنجره زیستی
- LH مثبت و BBT تأییدی؛ تعارض بیش از ۲ روز
- لنگر آنبوردینگ ۹۰ روز قبل؛ پریود رد شد؛ ویرایش سیکل ۳ ماه پیش
- قرص هورمونی؛ postpartum بدون پریود؛ سن ۴۶؛ سن ۱۶
- ثبت ساعت ۰۰:۳۰ به وقت تهران با سرور UTC
- موتور پیام: safety بر هر اولویتی غلبه دارد؛ requires_user_input فقط safety و engine؛ cooldown؛ تخصیص ثابت A/B
- خانواده: پیوند دوطرفه مستقل (لغو یکی دیگری را تغییر نمی‌دهد)؛ پذیرش بدون اشتراک متقابل؛ check() برای هر سطح؛ پیشنهاد دارو بدون تأیید اعمال نشود؛ سقف یادآوری ۳ در روز؛ غیرفعال شدن دامنه‌های بارداری بعد از زایمان؛ حذف پروفایل کودک با تأیید هر دو والد؛ هشدار فوری فقط با سوییچ و بدون جزئیات
- آستانه‌های §۳.۲: ۱۱۹/۷۹، ۱۲۰/۷۹، ۱۳۰/۷۹، ۱۲۹/۸۰، ۱۸۱/۱۰۰؛ قند ناشتای ۹۹، ۱۰۰، ۱۲۶؛ EPDS سؤال ۱۰ = ۱ با نمره کل ۴
### ۶.۲ منابع
- Wilcox AJ, Weinberg CR, Baird DD. Timing of sexual intercourse in relation to ovulation. NEJM 1995;333:1517–21.
- Munro MG, et al. FIGO 2018 Systems for abnormal uterine bleeding. Int J Gynaecol Obstet 2018;143:393–408.
- ACOG Committee Opinions 651 (adolescents), 700 (due date), 736 (postpartum care); Practice Bulletins 222 (hypertension in pregnancy), 179 (breast screening, self-awareness), 189 (nausea and vomiting of pregnancy).
- IOM 2009 Weight gain during pregnancy; IOM 2011 Dietary reference intakes for calcium and vitamin D.
- Cox JL, Holden JM, Sagovsky R. EPDS. Br J Psychiatry 1987;150:782–6.
- Whelton PK, et al. 2017 ACC/AHA high blood pressure guideline. Hypertension 2018;71:e13–e115; ACC/AHA 2025 guideline.
- American Heart Association — resting heart rate.
- ADA Standards of Care in Diabetes 2025, sections 2, 6, 15.
- WHO — BMI; haemoglobin cutoffs (2024); ferritin (2020); Child Growth Standards.
- Alexander EK, et al. 2017 ATA guidelines for thyroid disease in pregnancy. Thyroid 2017;27:315–89.
- USPSTF — cervical (2018), breast (2024), hypertension (2021), prediabetes/diabetes (2021).
- Zubler JM, et al. CDC developmental milestones. Pediatrics 2022;149:e2021052138.
- وزارت بهداشت ایران — برنامه کشوری ایمن‌سازی (آخرین نسخه).
