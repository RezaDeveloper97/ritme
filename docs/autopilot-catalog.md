# Autopilot catalog

> Generated from `frontend/src/app/autopilot/catalog.ts` — do not edit. Regenerate with `npm run autopilot:catalog` (in `frontend/`).
> Contract and rules: `docs/autopilot.md`. Members of a `family:*` come from data (the log taxonomy) and are listed in the live manifest (`/dev/autopilot`).

## Routes (`open` → `route`)

| Route | Title (fa) | Module |
|---|---|---|
| `/home` | امروز | — |
| `/calendar` | تقویم | — |
| `/cycle` | چرخه و سابقه‌ی پریود | — |
| `/log` | ثبت روزانه | — |
| `/log/customize` | شخصی‌سازی ثبت | — |
| `/profile` | من | — |
| `/profile/appearance` | ظاهر برنامه | — |
| `/profile/notifications` | اعلان‌ها | — |
| `/profile/credits` | مصرف اعتبار | — |
| `/plus/credits` | خرید اعتبار | — |
| `/services` | خدمات | — |
| `/analysis` | تحلیل | — |
| `/reminders` | یادآورها | reminders |
| `/checkups` | چکاپ‌ها | — |
| `/vitals` | علائم حیاتی | — |
| `/todo` | کارها | todo |
| `/labs` | آزمایش‌ها | — |
| `/record` | پرونده‌ی سلامت | record |
| `/learn` | یادگیری | learning |
| `/assistant` | دستیار | — |
| `/search` | جست‌وجو | — |
| `/fertility/log` | ثبت باروری | ttc |
| `/fertility/bbt` | نمودار دمای پایه | ttc |
| `/pregnancy` | بارداری | — |
| `/pregnancy/log` | ثبت بارداری | — |
| `/menopause/log` | ثبت علائم یائسگی | menopause |
| `/children` | کودکان | children |
| `/cycle/settings` | تنظیمات دوره | — |
| `/cycle/symptoms` | الگوی علائم | — |
| `/fertility/insights` | بینش باروری | ttc |
| `/analysis/body` | تحلیل بدن | — |
| `/analysis/correlations` | همبستگی‌ها | — |
| `/analysis/cycle` | تحلیل دوره | — |
| `/analysis/labs` | تحلیل آزمایش‌ها | — |
| `/analysis/period` | تحلیل پریود | — |
| `/analysis/symptoms` | تحلیل علائم | — |
| `/analysis/fertility` | تحلیل باروری | ttc |
| `/pregnancy/calendar` | تقویم بارداری | — |
| `/pregnancy/setup` | تنظیم بارداری | — |
| `/pregnancy/kicks` | شمارش ضربه | — |
| `/pregnancy/contractions` | زمان‌سنج انقباض | — |
| `/postpartum` | بعد از زایمان | postpartum |
| `/postpartum/setup` | تنظیم بعد از زایمان | postpartum |
| `/postpartum/recovery` | بهبودی بعد از زایمان | postpartum |
| `/children/new` | افزودن بچه | children |
| `/children/feeding` | شیردهی و خواب بچه | children |
| `/loss` | از دست دادن بارداری | — |
| `/loss/care` | مراقبت بعد از سقط | — |
| `/loss/next` | قدم بعدی بعد از سقط | — |
| `/teen/onboarding` | شروع نوجوان | teen |
| `/teen/parent` | همراهی مادر | teen |
| `/reminders/medication/new` | یادآور دارو | — |
| `/reminders/appointment/new` | یادآور نوبت دکتر | — |
| `/checkups/history` | تاریخچه‌ی چکاپ‌ها | — |
| `/checkups/self-exam` | خودآزمایی | — |
| `/checkups/custom/new` | چکاپ دلخواه | — |
| `/vitals/bp` | گزارش فشار خون | — |
| `/vitals/heart-rate` | گزارش ضربان قلب | — |
| `/vitals/glucose` | گزارش قند خون | — |
| `/vitals/bp/new` | ثبت فشار خون | — |
| `/vitals/heart-rate/new` | ثبت ضربان قلب | — |
| `/vitals/glucose/new` | ثبت قند خون | — |
| `/labs/new` | آپلود آزمایش | — |
| `/record/emergency` | کارت اورژانسی | — |
| `/record/export` | گزارش برای دکتر | record |
| `/record/export/preview` | پیش‌نمایش گزارش | record |
| `/record/timeline` | اسناد پرونده | record |
| `/record/share` | اشتراک پرونده | record |
| `/menopause/report` | گزارش یائسگی | menopause |
| `/profile/backup` | پشتیبان‌گیری | — |
| `/profile/devices` | دستگاه‌ها | devices |
| `/menopause/alert` | هشدار یائسگی | menopause |
| `/menopause/hot-flash` | گرگرفتگی | menopause |
| `/menopause/score` | امتیاز علائم یائسگی | menopause |
| `/menopause/score/questionnaire` | پرسش‌نامه‌ی علائم یائسگی | menopause |
| `/menopause/stage` | مرحله‌ی یائسگی | menopause |
| `/menopause/treatment` | درمان و مراقبت یائسگی | menopause |
| `/programs` | برنامه‌های درمانی | programs |
| `/programs/bleeding` | نمودار خونریزی | programs |
| `/programs/pain` | دفتر درد | programs |
| `/programs/pmdd` | پیگیری PMDD | programs |
| `/programs/pelvic` | کف لگن | programs |
| `/programs/pelvic/kegel` | تمرین کگل | programs |
| `/contraception` | پیشگیری از بارداری | contraception |
| `/contraception/missed` | قرص جاافتاده | contraception |
| `/contraception/other` | یادآورهای پیشگیری | contraception |
| `/contraception/setup` | انتخاب روش پیشگیری | contraception |
| `/ivf` | IVF | ivf |
| `/ivf/cycle` | ویرایش سیکل IVF | ivf |
| `/ivf/cycle/new` | سیکل جدید IVF | ivf |
| `/ivf/meds` | داروهای IVF | ivf |
| `/ivf/meds/new` | داروی جدید IVF | ivf |
| `/ivf/scan` | سونوی IVF | ivf |
| `/ivf/tww` | انتظار دو هفته‌ای | ivf |
| `/profile/mode` | حالت زندگی | — |
| `/profile/account` | حساب من | — |
| `/profile/language` | زبان و تقویم | — |
| `/profile/privacy` | حریم خصوصی | — |
| `/profile/about` | درباره‌ی ریتمی | — |
| `/profile/support` | پشتیبانی | — |
| `/profile/legal` | قوانین و حریم خصوصی | — |
| `/plus` | ریتمی پلاس | — |
| `/plus/plans` | پلن‌های اشتراک | — |
| `/plus/manage` | مدیریت اشتراک | — |
| `/assistant/history` | سابقه‌ی دستیار | — |
| `/assistant/profile` | داده‌های دستیار | — |
| `/learn/downloads` | دانلودها | learning |
| `/companion` | خانه‌ی همدم | companion |
| `/companion/actions` | پیشنهادهای همدم | companion |
| `/companion/calendar` | تقویم همدم | companion |
| `/companion/links` | کد همدم | companion |
| `/companions` | همدم‌ها | companion |
| `/companions/new` | دعوت از همدم | companion |
| `/city-services` | مکان‌های مادر و کودک | mother_child |
| `/city-services/join` | ثبت مکان من | mother_child |
| `/profile/bookings` | نوبت‌های مکان‌ها | mother_child |
| `/services/doctors` | دکترها | doctors |
| `/services/bookings` | نوبت‌های دکتر | doctors |
| `/services/home-lab` | آزمایش در منزل | — |
| `/insurance` | بیمه | insurance |
| `/insurance/centers` | مراکز طرف قرارداد | insurance |
| `/insurance/claims` | خسارت‌های بیمه | insurance |
| `/insurance/claims/new` | ثبت خسارت | insurance |
| `/insurance/coverage` | پوشش بیمه | insurance |
| `/insurance/questionnaire` | پرسش‌نامه‌ی بیمه | insurance |
| `/insurance/status` | وضعیت درخواست بیمه | insurance |
| `/shop` | فروشگاه | shop |
| `/shop/cart` | سبد خرید | shop |
| `/shop/checklist` | چک‌لیست خرید | shop |
| `/profile/orders` | سفارش‌های من | shop |

## Sheets (`open` → `sheet`)

| Sheet | Title (fa) | `arg` | Module |
|---|---|---|---|
| `log` | ثبت روزانه | YYYY-MM-DD (optional, today when absent) | — |
| `notifications` | اعلان‌ها | — | — |
| `todo-add` | کار تازه | — | todo |
| `vitals-add` | ثبت علائم حیاتی | — | — |
| `phase` | جزئیات فاز | — | — |
| `reminders-add` | افزودن یادآور | — | — |
| `checkup-mark-done` | ثبت انجام چکاپ | "<typeId>" or "<typeId>-<recordId>" | — |
| `lab-consent` | رضایت تحلیل آزمایش | upload (optional) | — |
| `articles` | مقاله‌ها | — | home_articles |
| `personal` | اطلاعات شخصی | — | — |
| `health` | اطلاعات سلامت | — | — |
| `reminders` | یادآورها | — | reminders |
| `language` | زبان | — | — |

## Targets

| Id | Value | Title (fa) | Surface | Reveal | Options |
|---|---|---|---|---|---|
| `log.save` | action | ذخیره‌ی ثبت روزانه | sheet `log` | — | — |
| `log.cat:*` | section | — | sheet `log` | — | — |
| `log.more:*` | section | — | sheet `log` | yes | — |
| `log.param:*` | single | — | sheet `log` | yes | — |
| `log.level:*` | single | — | sheet `log` | yes | — |
| `log.date:*` | action | — | sheet `log` | — | — |
| `nav.log` | action | دکمه‌ی ثبت | — | — | — |
| `nav.tab:*` | action | — | — | — | — |
| `sheet.close` | action | بستن پنجره | — | — | — |
| `log.panel:bleeding` | action | جزئیات خونریزی | sheet `log` | yes | — |
| `log.panel:pain` | action | جزئیات درد | sheet `log` | yes | — |
| `log.panel:measure` | action | جزئیات اندازه‌گیری‌ها | sheet `log` | yes | — |
| `log.panel-confirm` | action | تأیید یا ذخیره‌ی جزئیات | sheet `log` | — | — |
| `log.tile:*` | action | — | sheet `log` | — | — |
| `log.tab:*` | action | — | sheet `log` | — | — |
| `log.search` | text | جست‌وجوی ثبت روزانه | sheet `log` | — | — |
| `log.preset-row:*` | action | — | sheet `log` | — | — |
| `log.preset-group:*` | section | — | sheet `log` | — | — |
| `log.preset-bleeding` | single | خونریزی (یائسگی) | sheet `log` | — | — |
| `log.preset-triggers` | multi | محرک‌ها | sheet `log` | — | — |
| `log.preset-voice` | action | با صدا بگو (ثبت علائم) | sheet `log` | — | — |
| `log.panel-param:bleeding.flow` | single | شدت خونریزی (جزئیات) | sheet `log` | yes | — |
| `log.panel-param:bleeding.spotting` | bool | لکه‌بینی | sheet `log` | yes | — |
| `log.panel-param:bleeding.color` | single | رنگ خونریزی | sheet `log` | yes | — |
| `log.panel-param:bleeding.clots` | single | لخته | sheet `log` | yes | `none` `small` `large` |
| `log.panel-param:bleeding.odor` | single | بوی خونریزی | sheet `log` | yes | — |
| `log.panel-param:pain.score` | single | شدت درد (۱ تا ۱۰) | sheet `log` | yes | `1` `2` `3` `4` `5` `6` `7` `8` `9` `10` |
| `log.panel-param:pain.relief` | multi | چی دردت رو کم کرد | sheet `log` | yes | — |
| `log.panel-param:measurements.weight` | number | وزن | sheet `log` | yes | — |
| `log.panel-param:measurements.bbt` | number | دمای پایه‌ی بدن (ثبت روزانه) | sheet `log` | yes | — |
| `log.panel-param:measurements.lh_test` | single | تست LH | sheet `log` | yes | — |
| `log.panel-param:measurements.pregnancy_test` | single | تست بارداری | sheet `log` | yes | — |
| `period.toggle` | action | شروع یا پایان پریود | — | — | — |
| `period.day:*` | bool | — | — | — | — |
| `period.save` | action | ذخیره‌ی تاریخ‌های پریود | — | — | — |
| `period.spotting` | action | فقط لکه‌بینی است | — | — | — |
| `period.month-prev` | action | ماه قبل (ویرایش پریود) | — | — | — |
| `period.month-next` | action | ماه بعد (ویرایش پریود) | — | — | — |
| `customize.show:*` | bool | — | route `/log/customize` | — | — |
| `customize.pin:*` | bool | — | route `/log/customize` | — | — |
| `customize.save` | action | ذخیره‌ی شخصی‌سازی | route `/log/customize` | — | — |
| `customize.discard` | action | لغو تغییرات شخصی‌سازی | route `/log/customize` | — | — |
| `customize.add` | action | افزودن مورد دلخواه | route `/log/customize` | — | — |
| `customize.add-label` | text | نام مورد دلخواه | route `/log/customize` | yes | — |
| `customize.add-category` | single | دسته‌ی مورد دلخواه | route `/log/customize` | yes | — |
| `customize.add-submit` | action | ثبت مورد دلخواه | route `/log/customize` | yes | — |
| `fertility.lh` | single | نتیجه‌ی تست LH | route `/fertility/log` | — | `none` `negative` `faint` `positive` |
| `fertility.bbt` | number | دمای پایه‌ی بدن | route `/fertility/log` | — | — |
| `fertility.bbt-down` | action | کم‌کردن دمای پایه | route `/fertility/log` | — | — |
| `fertility.bbt-up` | action | زیاد‌کردن دمای پایه | route `/fertility/log` | — | — |
| `fertility.mucus` | single | ترشحات دهانه‌ی رحم | route `/fertility/log` | — | `dry` `sticky` `creamy` `egg_white` |
| `fertility.intercourse` | single | رابطه‌ی جنسی | route `/fertility/log` | — | `none` `unprotected` `protected` |
| `fertility.symptoms` | multi | علائم باروری | route `/fertility/log` | — | `ovarian_pain` `bloating` `breast_sensitivity` `spotting` |
| `fertility.note` | text | یادداشت روز | route `/fertility/log` | — | — |
| `fertility.save` | action | ذخیره‌ی ثبت باروری | route `/fertility/log` | — | — |
| `fertility.bbt-log` | action | ثبت دمای امروز | route `/fertility/bbt` | — | — |
| `fertility.bbt-insights` | action | باز کردن بینش باروری | route `/fertility/bbt` | — | — |
| `fertility.bbt-range:*` | action | — | route `/fertility/bbt` | — | — |
| `fertility.bbt-reminder` | bool | یادآور روزانه‌ی دمای پایه | route `/fertility/bbt` | — | — |
| `fertility.insights-log` | action | ثبت روز باروری | route `/fertility/insights` | — | — |
| `fertility.insights-tips` | action | نکته‌های بهتر شدن دقت | route `/fertility/insights` | — | — |
| `cycle-settings.auto` | bool | محاسبه‌ی خودکار طول دوره | route `/cycle/settings` | — | — |
| `cycle-settings.cycle-length` | integer | طول دوره | route `/cycle/settings` | — | — |
| `cycle-settings.period-length` | integer | طول پریود | route `/cycle/settings` | — | — |
| `cycle-settings.days-before` | integer | چند روز قبل از پریود یادآوری شود | route `/cycle/settings` | yes | — |
| `cycle-settings.reminder:*` | bool | — | route `/cycle/settings` | — | — |
| `cycle-settings.reminder-edit:*` | section | — | route `/cycle/settings` | — | — |
| `cycle-settings.reminder-time:*` | time | — | route `/cycle/settings` | yes | — |
| `cycle-settings.mode:*` | action | — | route `/cycle/settings` | — | — |
| `cycle.group:*` | action | — | route `/cycle/symptoms` | — | — |
| `cycle.add-previous` | action | افزودن دوره‌ی قبلی | route `/cycle` | — | — |
| `cycle.symptoms` | action | الگوی علائم | route `/cycle` | — | — |
| `phase-details.tab:*` | action | — | sheet `phase` | — | — |
| `calendar.day:*` | action | — | route `/calendar` | — | — |
| `calendar.month-prev` | action | ماه قبل | route `/calendar` | — | — |
| `calendar.month-next` | action | ماه بعد | route `/calendar` | — | — |
| `calendar.full-month` | bool | نمای ماه کامل | route `/calendar` | — | — |
| `calendar.edit-period` | action | ویرایش تاریخ پریود | route `/calendar` | — | — |
| `calendar.edit-this-period` | action | ویرایش این پریود | route `/calendar` | — | — |
| `calendar.add-to-period` | action | افزودن این روز به پریود | route `/calendar` | — | — |
| `calendar.start-period-here` | action | پریودم از این روز شروع شد | route `/calendar` | — | — |
| `calendar.log-day` | action | ثبت یا ویرایش روز انتخاب‌شده | route `/calendar` | — | — |
| `calendar.vitals` | action | علائم حیاتی | route `/calendar` | — | — |
| `home.today-tasks` | action | کارهای امروز | route `/home` | — | — |
| `home.later-vitals` | action | ثبت علائم حیاتی | route `/home` | — | — |
| `home.later-appointment` | action | ثبت وقت پزشک | route `/home` | — | — |
| `home.later-task` | action | افزودن کار | route `/home` | — | — |
| `home.later-feeding` | action | ثبت شیردهی | route `/home` | — | — |
| `home.reminders-all` | action | همه‌ی یادآورها | route `/home` | — | — |
| `home.reminder-add` | action | افزودن یادآور | route `/home` | — | — |
| `home.dose:*` | bool | — | route `/home` | — | — |
| `home.ttc-tile:*` | action | — | route `/home` | — | — |
| `home.day:*` | action | — | route `/home` | — | — |
| `home.phase-more` | action | جزئیات فاز | route `/home` | — | — |
| `home.phase-action` | action | اقدام کارت فاز | route `/home` | — | — |
| `home.log-today` | action | ثبت امروز | route `/home` | — | — |
| `home.period-started` | action | پریودم شروع شد | route `/home` | — | — |
| `home.period-not-yet` | action | هنوز نه | route `/home` | — | — |
| `home.period-ended` | action | پریودم تموم شد | route `/home` | — | — |
| `home.period-still-going` | action | هنوز ادامه داره | route `/home` | — | — |
| `home.edit-period` | action | ویرایش پریود | route `/home` | — | — |
| `home.log-card:*` | action | — | route `/home` | — | — |
| `analysis.card:*` | action | — | route `/analysis` | — | — |
| `analysis.report` | action | گزارش ماهانه | route `/analysis` | — | — |
| `analysis.range:*` | action | — | route `/analysis` | — | — |
| `analysis.filter` | single | فیلتر دسته‌بندی تحلیل | route `/analysis` | — | — |
| `analysis.tab:*` | action | — | route `/analysis` | — | — |
| `analysis.log` | action | ثبت روزانه (از تحلیل) | — | — | — |
| `analysis.month-prev` | action | گزارش ماه قبل | — | — | — |
| `analysis.month-next` | action | گزارش ماه بعد | — | — | — |
| `analysis.symptom-pick` | multi | انتخاب علائم نمودار | route `/analysis/symptoms` | — | — |
| `analysis.cycle-prev` | action | دوره‌ی قبلی | route `/analysis/fertility` | — | — |
| `analysis.cycle-next` | action | دوره‌ی بعدی | route `/analysis/fertility` | — | — |
| `analysis.labs-open` | action | باز کردن آزمایش‌ها | route `/analysis/labs` | — | — |
| `log.panel-param:*` | single | — | sheet `log` | — | — |
| `preg.open:log` | action | ثبت امروز بارداری | route `/pregnancy` | — | — |
| `preg.open:weekly` | action | چکاپ هفتگی | route `/pregnancy` | — | — |
| `preg.open:weeks` | action | هفته‌های بارداری | route `/pregnancy` | — | — |
| `preg.open:alerts` | action | هشدارهای بارداری | route `/pregnancy` | — | — |
| `preg.open:calendar` | action | تقویم و ویزیت‌ها | route `/pregnancy` | — | — |
| `preg.open:next-visit` | action | ویزیت بعدی | route `/pregnancy` | — | — |
| `preg.open:movement` | action | ثبت حرکت جنین | route `/pregnancy/log?tab=day` | — | — |
| `preg.delivered` | action | زایمان کردم | route `/pregnancy` | — | — |
| `preg.setup-edit` | action | ویرایش تاریخ بارداری | route `/pregnancy` | — | — |
| `preg.setup` | action | شروع تنظیم بارداری | route `/pregnancy` | — | — |
| `preg.retry` | action | تلاش دوباره | route `/pregnancy` | — | — |
| `preg.task:*` | bool | — | — | — | — |
| `preg.week:*` | action | — | — | — | — |
| `preg.week-tab:*` | action | — | — | — | — |
| `preg.day-picker` | action | انتخاب روز | route `/pregnancy/log?tab=day` | — | — |
| `preg.day-calendar` | date | تاریخ ثبت | route `/pregnancy/log?tab=day` | yes | — |
| `preg.mood` | single | حال امروز | route `/pregnancy/log?tab=day` | — | `5` `4` `3` `2` `1` |
| `preg.symptoms` | multi | علائم امروز | route `/pregnancy/log?tab=day` | — | `nausea` `vomiting` `fatigue` `headache` `back_pain` `breast_pain` `heartburn` `constipation` |
| `preg.severity:*` | single | — | — | — | — |
| `preg.water-less` | action | یک لیوان آب کمتر | route `/pregnancy/log?tab=day` | — | — |
| `preg.water-more` | action | یک لیوان آب بیشتر | route `/pregnancy/log?tab=day` | — | — |
| `preg.weight` | text | وزن | route `/pregnancy/log?tab=day` | — | — |
| `preg.spotting` | bool | لکه‌بینی | route `/pregnancy/log?tab=day` | — | — |
| `preg.note` | text | یادداشت روز | route `/pregnancy/log?tab=day` | — | — |
| `preg.save` | action | ذخیره‌ی ثبت امروز | route `/pregnancy/log?tab=day` | — | — |
| `preg.symptom:*` | bool | — | — | — | — |
| `preg.symptom-severity:*` | single | — | — | — | — |
| `preg.v1-notes` | text | یادداشت علائم | route `/pregnancy/log?tab=symptoms` | — | — |
| `preg.v1-save` | action | ذخیره‌ی علائم | route `/pregnancy/log?tab=symptoms` | — | — |
| `preg.weekly-weight` | number | وزن هفتگی | route `/pregnancy/log?tab=weekly` | — | — |
| `preg.weekly-swelling` | bool | ورم | route `/pregnancy/log?tab=weekly` | — | — |
| `preg.weekly-swelling-where` | multi | محل ورم | route `/pregnancy/log?tab=weekly` | yes | `feet` `hands` `face` |
| `preg.weekly-breath` | bool | تنگی نفس | route `/pregnancy/log?tab=weekly` | — | — |
| `preg.weekly-bp-device` | bool | دستگاه فشارخون دارم | route `/pregnancy/log?tab=weekly` | — | — |
| `preg.weekly-systolic` | number | فشار بالا | route `/pregnancy/log?tab=weekly` | yes | — |
| `preg.weekly-diastolic` | number | فشار پایین | route `/pregnancy/log?tab=weekly` | yes | — |
| `preg.weekly-fasting-sugar` | number | قند ناشتا | route `/pregnancy/log?tab=weekly` | — | — |
| `preg.weekly-post-meal-sugar` | number | قند بعد از غذا | route `/pregnancy/log?tab=weekly` | — | — |
| `preg.weekly-mood` | single | حال کلی این هفته | route `/pregnancy/log?tab=weekly` | — | `good` `moderate` `poor` |
| `preg.weekly-mental:*` | bool | — | — | — | — |
| `preg.weekly-mental-severity:*` | single | — | — | — | — |
| `preg.weekly-notes` | text | یادداشت هفتگی | route `/pregnancy/log?tab=weekly` | — | — |
| `preg.weekly-save` | action | ذخیره‌ی چکاپ هفتگی | route `/pregnancy/log?tab=weekly` | — | — |
| `preg.movement-status` | single | وضعیت حرکت جنین | route `/pregnancy/log?tab=movement` | — | — |
| `preg.movement-count` | number | تعداد حرکت | route `/pregnancy/log?tab=movement` | — | — |
| `preg.movement-first` | time | ساعت اولین حرکت | route `/pregnancy/log?tab=movement` | — | — |
| `preg.movement-last` | time | ساعت آخرین حرکت | route `/pregnancy/log?tab=movement` | — | — |
| `preg.movement-notes` | text | یادداشت حرکت جنین | route `/pregnancy/log?tab=movement` | — | — |
| `preg.movement-save` | action | ذخیره‌ی حرکت جنین | route `/pregnancy/log?tab=movement` | — | — |
| `preg.kick-tap` | action | ثبت یک ضربه | route `/pregnancy/kicks` | — | — |
| `preg.kick-undo` | action | برگردوندن آخرین ضربه | route `/pregnancy/kicks` | — | — |
| `preg.kick-stop` | action | پایان شمارش ضربه | route `/pregnancy/kicks` | — | — |
| `preg.contraction-toggle` | action | شروع/پایان انقباض | route `/pregnancy/contractions` | — | — |
| `preg.contraction-finish` | action | پایان ثبت انقباض‌ها | route `/pregnancy/contractions` | — | — |
| `preg.cal-prev` | action | ماه قبل | route `/pregnancy/calendar` | — | — |
| `preg.cal-next` | action | ماه بعد | route `/pregnancy/calendar` | — | — |
| `preg.cal-day:*` | action | — | — | — | — |
| `preg.new-visit` | action | ویزیت جدید | route `/pregnancy/calendar` | — | — |
| `preg.new-visit-day` | action | ویزیت جدید برای این روز | route `/pregnancy/calendar` | — | — |
| `preg.visit-stage` | action | رفتن به مرحله‌ی بعدی ویزیت | route `/pregnancy/calendar` | — | — |
| `preg.visit-result` | text | نتیجه‌ی ویزیت | route `/pregnancy/calendar` | yes | — |
| `preg.visit-result-save` | action | ذخیره‌ی نتیجه‌ی ویزیت | route `/pregnancy/calendar` | yes | — |
| `preg.report` | action | گزارش بارداری | route `/pregnancy/calendar` | — | — |
| `preg.setup-basis` | single | مبنای محاسبه‌ی بارداری | route `/pregnancy/setup` | — | `lmp` `ultrasound` `due` `week` |
| `preg.setup-date-open:lmp` | action | باز کردن تاریخ آخرین پریود | route `/pregnancy/setup` | yes | — |
| `preg.setup-date:lmp` | date | اولین روز آخرین پریود | route `/pregnancy/setup` | yes | — |
| `preg.setup-date-open:scan` | action | باز کردن تاریخ سونوگرافی | route `/pregnancy/setup` | yes | — |
| `preg.setup-date:scan` | date | تاریخ سونوگرافی | route `/pregnancy/setup` | yes | — |
| `preg.setup-date-open:due` | action | باز کردن تاریخ زایمان | route `/pregnancy/setup` | yes | — |
| `preg.setup-date:due` | date | تاریخ احتمالی زایمان | route `/pregnancy/setup` | yes | — |
| `preg.setup-weeks:ultrasound` | integer | هفته در سونوگرافی | route `/pregnancy/setup` | yes | — |
| `preg.setup-days:ultrasound` | integer | روز در سونوگرافی | route `/pregnancy/setup` | yes | — |
| `preg.setup-weeks:week` | integer | هفته‌ی فعلی بارداری | route `/pregnancy/setup` | yes | — |
| `preg.setup-days:week` | integer | روز فعلی بارداری | route `/pregnancy/setup` | yes | — |
| `preg.setup-next` | action | ادامه‌ی مراحل تنظیم | route `/pregnancy/setup` | — | — |
| `preg.setup-skip` | action | رد کردن سوابق | route `/pregnancy/setup` | yes | — |
| `preg.setup-miscarriage` | bool | سابقه‌ی سقط | route `/pregnancy/setup` | yes | — |
| `preg.setup-high-risk` | bool | بارداری پرخطر | route `/pregnancy/setup` | yes | — |
| `preg.setup-conditions` | multi | بیماری‌های زمینه‌ای | route `/pregnancy/setup` | yes | `none` `chronic_hypertension` `diabetes` `hypothyroidism` `hyperthyroidism` |
| `preg.setup-blood` | single | گروه خونی | route `/pregnancy/setup` | yes | `A` `B` `AB` `O` |
| `preg.setup-rh` | single | آر‌اچ | route `/pregnancy/setup` | yes | `positive` `negative` |
| `preg.setup-finish` | action | ثبت و پایان تنظیم | route `/pregnancy/setup` | yes | — |
| `preg.setup-continue` | action | ادامه به بارداری | route `/pregnancy/setup` | — | — |
| `baby.feed-child` | single | انتخاب بچه | — | — | — |
| `baby.feed-type:*` | action | — | — | — | — |
| `baby.feed-side:*` | action | — | — | — | — |
| `baby.feed-start` | action | شروع تایمر شیردهی | — | — | — |
| `baby.feed-amount` | integer | مقدار شیر | — | — | — |
| `baby.feed-finish` | action | پایان و ذخیره‌ی شیردهی | — | — | — |
| `baby.feed-manual` | action | ثبت دستی شیردهی | — | — | — |
| `baby.feed-manual-time` | time | ساعت شیردهی | — | yes | — |
| `baby.feed-manual-left` | integer | دقیقه‌ی سینه‌ی چپ | — | yes | — |
| `baby.feed-manual-right` | integer | دقیقه‌ی سینه‌ی راست | — | yes | — |
| `baby.feed-manual-minutes` | integer | مدت شیردهی | — | yes | — |
| `baby.feed-manual-ml` | integer | مقدار شیر دستی | — | yes | — |
| `baby.feed-manual-save` | action | ذخیره‌ی شیردهی دستی | — | yes | — |
| `baby.sleep-start` | action | خوابید | — | — | — |
| `baby.sleep-stop` | action | بیدار شد | — | — | — |
| `baby.sleep-manual` | action | ثبت دستی خواب | — | — | — |
| `baby.sleep-from` | time | ساعت شروع خواب | — | yes | — |
| `baby.sleep-to` | time | ساعت پایان خواب | — | yes | — |
| `baby.sleep-manual-save` | action | ذخیره‌ی خواب دستی | — | yes | — |
| `baby.diaper:*` | action | — | — | — | — |
| `baby.today:*` | action | — | — | — | — |
| `pp.setup` | action | تنظیم پس از زایمان | route `/postpartum` | — | — |
| `pp.retry` | action | تلاش دوباره | route `/postpartum` | — | — |
| `pp.open:recovery` | action | ثبت بهبودی امروز | route `/postpartum` | — | — |
| `pp.mood` | multi | حال امروز بعد از زایمان | route `/postpartum` | — | `good` `tired` `restless` `sad` |
| `pp.breastfeeding` | bool | شیر می‌دم | route `/postpartum` | — | — |
| `pp.tile:*` | action | — | — | — | — |
| `pp.visit-today` | action | ثبت ویزیت امروز | route `/postpartum` | — | — |
| `pp.visit-add` | action | افزودن ویزیت | route `/postpartum` | — | — |
| `pp.child-vaccines:*` | action | — | — | — | — |
| `pp.birth-date` | date | تاریخ زایمان | route `/postpartum/setup` | — | — |
| `pp.delivery` | single | نوع زایمان | route `/postpartum/setup` | — | `vaginal` `cesarean` `none` |
| `pp.babies` | integer | تعداد نوزاد | route `/postpartum/setup` | — | — |
| `pp.setup-save` | action | فعال‌سازی پس از زایمان | route `/postpartum/setup` | — | — |
| `pp.rec-temp-add` | action | افزودن دمای بدن | route `/postpartum/recovery` | — | — |
| `pp.rec-temp` | number | دمای بدن | route `/postpartum/recovery` | yes | — |
| `pp.rec-temp-clear` | action | حذف دمای بدن | route `/postpartum/recovery` | yes | — |
| `pp.rec-flags` | multi | نشانه‌های خطر | route `/postpartum/recovery` | — | `padSoakedWithinHour` `severePain` `breathless` |
| `pp.rec-lochia` | single | مقدار خونریزی | route `/postpartum/recovery` | — | `none` `spotting` `light` `medium` `heavy` |
| `pp.rec-lochia-color` | single | رنگ خونریزی | route `/postpartum/recovery` | — | `red` `pink_brown` `yellow_white` |
| `pp.rec-pain` | single | شدت درد | route `/postpartum/recovery` | — | `none` `mild` `moderate` `severe` |
| `pp.rec-pain-where` | multi | محل درد | route `/postpartum/recovery` | yes | `stitches` `abdomen` `breast` `back` `head` |
| `pp.rec-feeds` | integer | تعداد شیردهی | route `/postpartum/recovery` | — | — |
| `pp.rec-breasts` | multi | وضعیت سینه‌ها | route `/postpartum/recovery` | — | `normal` `engorgement` `nipple_pain` `redness` |
| `pp.rec-sleep` | number | ساعت خواب | route `/postpartum/recovery` | — | — |
| `pp.rec-save` | action | ذخیره‌ی ثبت بهبودی | route `/postpartum/recovery` | — | — |
| `pp.visit-book` | action | رزرو ویزیت بعد از زایمان | route `/postpartum/recovery` | — | — |
| `child.open:*` | action | — | — | — | — |
| `child.add` | action | افزودن بچه | route `/children` | — | — |
| `child.retry` | action | تلاش دوباره | — | — | — |
| `child.edit` | action | ویرایش بچه | — | — | — |
| `child.add-measurement` | action | ثبت اندازه‌گیری | — | — | — |
| `child.vaccine-details` | action | جزئیات واکسن | — | — | — |
| `child.tile:vaccines` | action | واکسن‌ها | — | — | — |
| `child.tile:growth` | action | رشد | — | — | — |
| `child.tile:milestones` | action | نقاط عطف | — | — | — |
| `child.tile:learn` | action | آموزش | — | — | — |
| `child.form-name` | text | اسم بچه | route `/children/new` | — | — |
| `child.form-birth-open` | action | باز کردن تاریخ تولد | route `/children/new` | — | — |
| `child.form-birth` | date | تاریخ تولد | route `/children/new` | yes | — |
| `child.form-sex` | single | جنسیت | route `/children/new` | — | `girl` `boy` `none` |
| `child.form-weight` | number | وزن تولد | route `/children/new` | — | — |
| `child.form-length` | number | قد تولد | route `/children/new` | — | — |
| `child.form-head` | number | دور سر تولد | route `/children/new` | — | — |
| `child.form-delivery` | single | نوع زایمان بچه | route `/children/new` | — | `vaginal` `cesarean` `none` |
| `child.form-save` | action | ذخیره‌ی بچه | route `/children/new` | — | — |
| `child.growth-add` | action | افزودن اندازه‌گیری رشد | — | — | — |
| `child.growth-tab:*` | action | — | — | — | — |
| `child.growth-date-open` | action | باز کردن تاریخ اندازه‌گیری | — | yes | — |
| `child.growth-date` | date | تاریخ اندازه‌گیری | — | yes | — |
| `child.growth-weight` | number | وزن بچه | — | yes | — |
| `child.growth-length` | number | قد بچه | — | yes | — |
| `child.growth-head` | number | دور سر بچه | — | yes | — |
| `child.growth-save` | action | ذخیره‌ی اندازه‌گیری | — | yes | — |
| `child.learn-topic` | single | موضوع آموزش | — | — | — |
| `child.learn-all` | action | نمایش همه‌ی موضوع‌ها | — | — | — |
| `child.tip:*` | action | — | — | — | — |
| `child.ms-month` | single | سن نقاط عطف | — | — | — |
| `child.ms-item:*` | bool | — | — | — | — |
| `child.vaccine-tab:*` | action | — | — | — | — |
| `child.vaccine-dose:*` | bool | — | — | — | — |
| `child.vaccine-book:*` | action | — | — | — | — |
| `child.vaccine-mark:*` | action | — | — | — | — |
| `child.vaccine-date-open` | action | باز کردن تاریخ واکسن | — | — | — |
| `child.vaccine-date` | date | تاریخ تزریق | — | yes | — |
| `child.vaccine-note` | text | یادداشت واکسن | — | — | — |
| `child.vaccine-save` | action | ذخیره‌ی واکسن | — | — | — |
| `loss.retry` | action | تلاش دوباره | route `/loss` | — | — |
| `loss.type` | single | نوع از دست دادن بارداری | route `/loss` | — | `early_miscarriage` `late_miscarriage` `ectopic` `chemical` `unspecified` |
| `loss.date-open` | action | باز کردن تاریخ اتفاق | route `/loss` | — | — |
| `loss.date` | date | تاریخ اتفاق | route `/loss` | yes | — |
| `loss.date-save` | action | ذخیره‌ی تاریخ | route `/loss` | yes | — |
| `loss.tell` | bool | خبر دادن به همراه | route `/loss` | — | — |
| `loss.continue` | action | ادامه | route `/loss` | — | — |
| `loss.open:bleeding` | action | پیگیری خونریزی | route `/loss/care` | — | — |
| `loss.bleeding-log` | action | ثبت خونریزی امروز | route `/loss/care` | yes | — |
| `loss.bleeding-stopped` | action | خونریزی بند اومد | route `/loss/care` | yes | — |
| `loss.open:beta` | action | آزمایش بتا | route `/loss/care` | — | — |
| `loss.beta-date` | date | روز آزمایش بتا | route `/loss/care` | yes | — |
| `loss.beta-save` | action | ذخیره‌ی یادآور بتا | route `/loss/care` | yes | — |
| `loss.beta-negative` | action | جواب بتا منفی شد | route `/loss/care` | yes | — |
| `loss.open:visit` | action | ویزیت پیگیری | route `/loss/care` | — | — |
| `loss.visit-date` | date | روز ویزیت | route `/loss/care` | yes | — |
| `loss.visit-time` | single | ساعت ویزیت | route `/loss/care` | yes | — |
| `loss.visit-save` | action | ذخیره‌ی ویزیت | route `/loss/care` | yes | — |
| `loss.mood` | single | امروز چه حسی داری | route `/loss/care` | — | `sad` `numb` `angry` `a_bit_better` |
| `loss.open:note` | action | یادداشت خصوصی | route `/loss/care` | — | — |
| `loss.note` | text | متن یادداشت خصوصی | route `/loss/care` | yes | — |
| `loss.note-save` | action | ذخیره‌ی یادداشت | route `/loss/care` | yes | — |
| `loss.next` | action | ادامه به قدم بعدی | route `/loss/care` | — | — |
| `loss.next-choice` | single | قدم بعدی | route `/loss/next` | — | `cycle` `ttc` `nothing` |
| `loss.next-save` | action | ذخیره‌ی قدم بعدی | route `/loss/next` | — | — |
| `teen.age` | single | بازه‌ی سنی | route `/teen/onboarding` | — | `10_12` `13_15` `16_17` |
| `teen.menarche` | single | اولین پریود | route `/teen/onboarding` | — | `not_yet` `under_1y` `over_1y` |
| `teen.onboarding-continue` | action | ادامه | route `/teen/onboarding` | — | — |
| `teen.open:calendar` | action | تقویم پریود | — | — | — |
| `teen.open:parent` | action | همراهی مادر | — | — | — |
| `teen.kit:*` | bool | — | — | — | — |
| `teen.retry` | action | تلاش دوباره | — | — | — |
| `teen.share:*` | bool | — | — | — | — |
| `teen.invite-open` | action | دعوت مادر | route `/teen/parent` | — | — |
| `teen.invite-phone` | text | شماره‌ی مادر | route `/teen/parent` | yes | — |
| `teen.invite-name` | text | اسم مادر | route `/teen/parent` | yes | — |
| `teen.parent-note` | text | یادداشت برای مادر | route `/teen/parent` | — | — |
| `teen.parent-note-save` | action | ذخیره‌ی یادداشت برای مادر | route `/teen/parent` | — | — |
| `care.tab:all` | action | همه‌ی یادآورها | route `/reminders` | — | — |
| `care.tab:medications` | action | تب داروها | route `/reminders` | — | — |
| `care.tab:appointments` | action | تب نوبت‌ها | route `/reminders` | — | — |
| `care.notifications` | action | تنظیمات اعلان | route `/reminders` | — | — |
| `care.add` | action | یادآور جدید | route `/reminders` | — | — |
| `care.dose:*` | bool | — | route `/reminders` | — | — |
| `care.add:medication` | action | افزودن دارو | route `/reminders` | — | — |
| `care.add:appointment` | action | افزودن نوبت | route `/reminders` | — | — |
| `care.med:*` | action | — | route `/reminders` | — | — |
| `care.med-active:*` | bool | — | route `/reminders` | — | — |
| `care.appt:*` | action | — | route `/reminders` | — | — |
| `care.choice:medication` | action | دارو یا مکمل | sheet `reminders-add` | — | — |
| `care.choice:appointment` | action | نوبت دکتر | sheet `reminders-add` | — | — |
| `care.choice:consult` | action | مشاوره‌ی تلفنی | sheet `reminders-add` | — | — |
| `med.name` | text | نام دارو | route `/reminders/medication/new` | — | — |
| `med.start` | action | تاریخ شروع مصرف | route `/reminders/medication/new` | — | — |
| `med.frequency` | action | هر چند وقت یک بار | route `/reminders/medication/new` | — | — |
| `med.dose:*` | action | — | route `/reminders/medication/new` | — | — |
| `med.time:*` | action | — | route `/reminders/medication/new` | — | — |
| `med.meal:*` | single | — | route `/reminders/medication/new` | — | — |
| `med.dose-remove:*` | action | — | route `/reminders/medication/new` | — | — |
| `med.dose-add` | action | دوز دیگر | route `/reminders/medication/new` | — | — |
| `med.duration` | action | مدت مصرف | route `/reminders/medication/new` | — | — |
| `med.notify` | bool | یادآوری با اعلان | route `/reminders/medication/new` | — | — |
| `med.notes` | text | یادداشت دارو | route `/reminders/medication/new` | — | — |
| `med.save` | action | ذخیره‌ی یادآور دارو | route `/reminders/medication/new` | — | — |
| `med.date` | date | تاریخ شروع | route `/reminders/medication/new` | yes | — |
| `med.date-done` | action | تأیید تاریخ | route `/reminders/medication/new` | yes | — |
| `med.freq-every` | integer | عدد تکرار | route `/reminders/medication/new` | yes | — |
| `med.freq-unit` | single | واحد تکرار | route `/reminders/medication/new` | yes | `day` `hour` |
| `med.freq-save` | action | ذخیره‌ی تکرار | route `/reminders/medication/new` | yes | — |
| `med.dose-amount` | integer | مقدار دوز | route `/reminders/medication/new` | yes | — |
| `med.dose-unit` | single | واحد دوز | route `/reminders/medication/new` | yes | `pill` `capsule` `drop` `ml` `spoon` `puff` `injection` |
| `med.dose-save` | action | ذخیره‌ی دوز | route `/reminders/medication/new` | yes | — |
| `med.time-hour` | integer | ساعت مصرف | route `/reminders/medication/new` | yes | — |
| `med.time-minute` | integer | دقیقه‌ی مصرف | route `/reminders/medication/new` | yes | — |
| `med.time-done` | action | تأیید ساعت | route `/reminders/medication/new` | yes | — |
| `med.duration-choice` | single | نوع مدت مصرف | route `/reminders/medication/new` | yes | `ongoing` `until_date` `pregnancy_end` |
| `med.ends-on` | date | تاریخ پایان مصرف | route `/reminders/medication/new` | yes | — |
| `med.duration-done` | action | تأیید مدت مصرف | route `/reminders/medication/new` | yes | — |
| `appt.with` | text | نام دکتر | route `/reminders/appointment/new` | — | — |
| `appt.specialty` | text | تخصص | route `/reminders/appointment/new` | — | — |
| `appt.topic` | single | موضوع نوبت | route `/reminders/appointment/new` | — | `ultrasound` `checkup` `lab` `consult` `vaccine` `other` |
| `appt.title` | text | توضیح کوتاه | route `/reminders/appointment/new` | — | — |
| `appt.date` | action | تاریخ نوبت | route `/reminders/appointment/new` | — | — |
| `appt.time` | action | ساعت نوبت | route `/reminders/appointment/new` | — | — |
| `appt.location` | text | محل یا لینک | route `/reminders/appointment/new` | — | — |
| `appt.remind` | single | یادآوری قبل از نوبت | route `/reminders/appointment/new` | — | `1h` `3h` `1d` `2d` |
| `appt.prep` | text | چیزهایی که باید ببرم | route `/reminders/appointment/new` | — | — |
| `appt.save` | action | ذخیره‌ی نوبت | route `/reminders/appointment/new` | — | — |
| `appt.date-pick` | date | روز نوبت | route `/reminders/appointment/new` | yes | — |
| `appt.date-done` | action | تأیید روز نوبت | route `/reminders/appointment/new` | yes | — |
| `appt.hour` | integer | ساعت | route `/reminders/appointment/new` | yes | — |
| `appt.minute` | integer | دقیقه | route `/reminders/appointment/new` | yes | — |
| `appt.time-done` | action | تأیید ساعت نوبت | route `/reminders/appointment/new` | yes | — |
| `apptd.edit` | action | ویرایش نوبت | — | — | — |
| `apptd.remind` | bool | یادآوری نوبت | — | — | — |
| `apptd.prep:*` | bool | — | — | — | — |
| `apptd.calendar` | action | افزودن نوبت به تقویم | — | — | — |
| `ck.tab:all` | action | همه‌ی چکاپ‌ها | route `/checkups` | — | — |
| `ck.tab:action` | action | نیازمند اقدام | route `/checkups` | — | — |
| `ck.tab:done` | action | انجام‌شده | route `/checkups` | — | — |
| `ck.settings` | action | تنظیمات برنامه‌ی چکاپ | route `/checkups` | — | — |
| `ck.add-custom` | action | چکاپ دلخواه جدید | route `/checkups` | — | — |
| `ck.item:*` | action | — | route `/checkups` | — | — |
| `ck.enabled:*` | bool | — | route `/checkups` | yes | — |
| `ck.remind:*` | bool | — | route `/checkups` | yes | — |
| `ckd.remind` | bool | یادآوری چکاپ | — | — | — |
| `ckd.guide` | action | راهنمای چکاپ | — | — | — |
| `ckd.mark-done` | action | ثبت انجام چکاپ | — | — | — |
| `ckd.book` | action | رزرو نوبت چکاپ | — | — | — |
| `ckd.record:*` | action | — | — | — | — |
| `ckd.history` | action | همه‌ی سوابق چکاپ | — | — | — |
| `ckd.edit-custom` | action | ویرایش چکاپ دلخواه | — | — | — |
| `ckh.tab:all` | action | همه‌ی سوابق | route `/checkups/history` | — | — |
| `ckh.tab:this_year` | action | امسال | route `/checkups/history` | — | — |
| `ckh.tab:with_attachment` | action | با پیوست | route `/checkups/history` | — | — |
| `ckh.export` | action | خروجی PDF سوابق | route `/checkups/history` | — | — |
| `ckh.more` | action | سوابق بیشتر | route `/checkups/history` | — | — |
| `ckh.summary` | action | خلاصه‌ی چکاپ‌ها | route `/checkups/history` | — | — |
| `ckh.record:*` | action | — | route `/checkups/history` | — | — |
| `cks.findings` | multi | یافته‌های خودآزمایی | route `/checkups/self-exam` | — | — |
| `cks.done` | action | انجام دادم | route `/checkups/self-exam` | — | — |
| `cks.remind` | bool | یادآوری خودآزمایی | route `/checkups/self-exam` | — | — |
| `cks.history` | action | تاریخچه‌ی خودآزمایی | route `/checkups/self-exam` | — | — |
| `ckc.title` | text | نام چکاپ | route `/checkups/custom/new` | — | — |
| `ckc.interval` | single | فاصله‌ی چکاپ | route `/checkups/custom/new` | — | `1` `6` `12` `24` `36` |
| `ckc.performer` | single | چه کسی انجام می‌دهد | route `/checkups/custom/new` | — | `self` `doctor` `lab` `dentist` |
| `ckc.last-done` | action | آخرین بار انجام | route `/checkups/custom/new` | — | — |
| `ckc.note` | text | یادداشت چکاپ | route `/checkups/custom/new` | — | — |
| `ckc.save` | action | ذخیره‌ی چکاپ | route `/checkups/custom/new` | — | — |
| `ckc.date` | date | تاریخ آخرین انجام | route `/checkups/custom/new` | yes | — |
| `ckc.date-done` | action | تأیید تاریخ | route `/checkups/custom/new` | yes | — |
| `ckm.date` | action | تاریخ انجام | sheet `checkup-mark-done` | — | — |
| `ckm.test-kind` | single | نوع آزمایش | sheet `checkup-mark-done` | — | — |
| `ckm.result` | single | نتیجه‌ی چکاپ | sheet `checkup-mark-done` | — | `normal` `follow_up` `pending` |
| `ckm.note` | text | یادداشت نتیجه | sheet `checkup-mark-done` | — | — |
| `ckm.next` | action | تغییر موعد بعدی | sheet `checkup-mark-done` | — | — |
| `ckm.save` | action | ذخیره‌ی چکاپ | sheet `checkup-mark-done` | — | — |
| `ckm.calendar` | date | روز انجام | sheet `checkup-mark-done` | yes | — |
| `ckm.date-done` | action | تأیید روز | sheet `checkup-mark-done` | yes | — |
| `ckcard.all` | action | همه‌ی چکاپ‌ها | — | — | — |
| `ckcard.action:*` | action | — | — | — | — |
| `vt.record` | action | پرونده‌ی سلامت | route `/vitals` | — | — |
| `vt.add:bp` | action | ثبت فشار خون | route `/vitals` | — | — |
| `vt.add:hr` | action | ثبت ضربان قلب | route `/vitals` | — | — |
| `vt.add:glucose` | action | ثبت قند خون | route `/vitals` | — | — |
| `vt.latest:bp` | action | آخرین فشار خون | route `/vitals` | — | — |
| `vt.latest:hr` | action | آخرین ضربان قلب | route `/vitals` | — | — |
| `vt.latest:glucose` | action | آخرین قند خون | route `/vitals` | — | — |
| `vt.plan-edit` | action | برنامه‌ی اندازه‌گیری هفته | route `/vitals` | — | — |
| `vt.recent-all` | action | همه‌ی اندازه‌گیری‌ها | route `/vitals` | — | — |
| `vt.plan-type:*` | single | — | route `/vitals` | yes | — |
| `vt.plan-slot:*` | single | — | route `/vitals` | yes | — |
| `vt.plan-days:*` | multi | — | route `/vitals` | yes | — |
| `vt.plan-time:*` | time | — | route `/vitals` | yes | — |
| `vt.plan-add` | action | افزودن ردیف برنامه | route `/vitals` | yes | — |
| `vt.plan-save` | action | ذخیره‌ی برنامه | route `/vitals` | yes | — |
| `vt.targets-open` | action | هدف‌های قند | route `/vitals` | — | — |
| `vt.target-low:fasting` | integer | حداقل هدف قند ناشتا | route `/vitals` | yes | — |
| `vt.target-high:fasting` | integer | حداکثر هدف قند ناشتا | route `/vitals` | yes | — |
| `vt.target-by:fasting` | single | تعیین‌کننده‌ی هدف ناشتا | route `/vitals` | yes | `doctor` `user` |
| `vt.target-low:pre_meal` | integer | حداقل هدف قند قبل از غذا | route `/vitals` | yes | — |
| `vt.target-high:pre_meal` | integer | حداکثر هدف قند قبل از غذا | route `/vitals` | yes | — |
| `vt.target-by:pre_meal` | single | تعیین‌کننده‌ی هدف قبل از غذا | route `/vitals` | yes | `doctor` `user` |
| `vt.target-low:post_meal` | integer | حداقل هدف قند بعد از غذا | route `/vitals` | yes | — |
| `vt.target-high:post_meal` | integer | حداکثر هدف قند بعد از غذا | route `/vitals` | yes | — |
| `vt.target-by:post_meal` | single | تعیین‌کننده‌ی هدف بعد از غذا | route `/vitals` | yes | `doctor` `user` |
| `vt.target-low:bedtime` | integer | حداقل هدف قند موقع خواب | route `/vitals` | yes | — |
| `vt.target-high:bedtime` | integer | حداکثر هدف قند موقع خواب | route `/vitals` | yes | — |
| `vt.target-by:bedtime` | single | تعیین‌کننده‌ی هدف موقع خواب | route `/vitals` | yes | `doctor` `user` |
| `vt.targets-save` | action | ذخیره‌ی هدف‌ها | route `/vitals` | yes | — |
| `vt.choice:bp` | action | فشار خون | sheet `vitals-add` | — | — |
| `vt.choice:hr` | action | ضربان قلب | sheet `vitals-add` | — | — |
| `vt.choice:glucose` | action | قند خون | sheet `vitals-add` | — | — |
| `vt.choice:hub` | action | علائم حیاتی | sheet `vitals-add` | — | — |
| `vtadd.systolic` | integer | فشار بالا | — | — | — |
| `vtadd.diastolic` | integer | فشار پایین | — | — | — |
| `vtadd.pulse` | integer | نبض | — | — | — |
| `vtadd.arm` | single | دست اندازه‌گیری | — | — | `left` `right` |
| `vtadd.position` | single | حالت بدن | — | — | `sitting` `standing` `lying` |
| `vtadd.glucose` | number | مقدار قند خون | — | — | — |
| `vtadd.unit` | single | واحد قند | — | — | `mg_dl` `mmol_l` |
| `vtadd.glucose-context` | single | زمان اندازه‌گیری قند | — | — | `fasting` `before_meal` `after_meal` `bedtime` `random` |
| `vtadd.method` | single | روش اندازه‌گیری | — | — | `glucometer` `lab` `sensor` |
| `vtadd.bpm` | integer | ضربان قلب | — | — | — |
| `vtadd.hr-context` | single | حالت ضربان قلب | — | — | `resting` `after_exercise` `after_waking` `stress` |
| `vtadd.note` | text | یادداشت اندازه‌گیری | — | — | — |
| `vtadd.time` | action | زمان اندازه‌گیری | — | — | — |
| `vtadd.time-now` | action | همین حالا | — | yes | — |
| `vtadd.time-done` | action | تأیید زمان | — | yes | — |
| `vtadd.date` | date | روز اندازه‌گیری | — | yes | — |
| `vtadd.clock` | time | ساعت اندازه‌گیری | — | yes | — |
| `vtadd.save` | action | ذخیره‌ی اندازه‌گیری | — | — | — |
| `vtr.add` | action | ثبت اندازه‌گیری جدید | — | — | — |
| `vtr.share` | action | ارسال گزارش برای دکتر | — | — | — |
| `vtr.filter:all` | action | همه‌ی زمان‌ها | route `/vitals/glucose` | — | — |
| `vtr.filter:fasting` | action | ناشتا | route `/vitals/glucose` | — | — |
| `vtr.filter:after_meal` | action | بعد از غذا | route `/vitals/glucose` | — | — |
| `vtr.range-chips` | single | بازه‌ی گزارش قند | route `/vitals/glucose` | — | `14d` `30d` `90d` |
| `vtr.range:7d` | action | بازه‌ی ۷ روز | — | — | — |
| `vtr.range:30d` | action | بازه‌ی ۳۰ روز | — | — | — |
| `vtr.range:90d` | action | بازه‌ی ۹۰ روز | — | — | — |
| `vtr.more` | action | اندازه‌گیری‌های بیشتر | — | — | — |
| `todo.filter` | single | دسته‌ی کارها | route `/todo` | — | `all` `shopping` `work` `personal` `health` |
| `todo.new` | action | کار جدید | route `/todo` | — | — |
| `todo.done-section` | section | کارهای انجام‌شده | route `/todo` | — | — |
| `todo.suggest-add` | action | افزودن پیشنهاد | route `/todo` | — | — |
| `todo.suggest-dismiss` | action | رد پیشنهاد | route `/todo` | — | — |
| `todo.task:*` | bool | — | — | — | — |
| `todo.open:*` | action | — | — | — | — |
| `todo.item:*` | bool | — | — | — | — |
| `tdadd.title` | text | عنوان کار | sheet `todo-add` | — | — |
| `tdadd.note` | text | یادداشت کار | sheet `todo-add` | — | — |
| `tdadd.today` | bool | امروز | sheet `todo-add` | — | — |
| `tdadd.tomorrow` | bool | فردا | sheet `todo-add` | — | — |
| `tdadd.date` | action | انتخاب تاریخ | sheet `todo-add` | — | — |
| `tdadd.time-open` | section | ساعت کار | sheet `todo-add` | — | — |
| `tdadd.time` | time | ساعت انجام | sheet `todo-add` | yes | — |
| `tdadd.category-open` | section | دسته‌ی کار | sheet `todo-add` | — | — |
| `tdadd.category` | single | دسته‌ی کار | sheet `todo-add` | yes | `shopping` `work` `personal` `health` |
| `tdadd.remind` | bool | یادآوری کار | sheet `todo-add` | — | — |
| `tdadd.save` | action | ذخیره‌ی کار | sheet `todo-add` | — | — |
| `tdadd.calendar` | date | تاریخ کار | sheet `todo-add` | yes | — |
| `tdadd.clear-date` | action | بدون تاریخ | sheet `todo-add` | yes | — |
| `tdl.menu` | action | منوی لیست | — | — | — |
| `tdl.done-section` | section | موارد انجام‌شده‌ی لیست | — | — | — |
| `tdl.add-open` | action | افزودن مورد به لیست | — | — | — |
| `tdl.add-title` | text | عنوان مورد لیست | — | yes | — |
| `tdl.add-save` | action | ذخیره‌ی مورد | — | yes | — |
| `tdl.edit` | action | ویرایش لیست | — | yes | — |
| `lab.about` | action | درباره‌ی تحلیل آزمایش | route `/labs` | — | — |
| `lab.upload` | action | آپلود آزمایش | route `/labs` | — | — |
| `lab.history-toggle` | section | همه‌ی آزمایش‌ها | route `/labs` | — | — |
| `lab.row:*` | action | — | route `/labs` | — | — |
| `labu.category` | single | نوع آزمایش | route `/labs/new` | — | — |
| `labu.date` | action | تاریخ آزمایش | route `/labs/new` | — | — |
| `labu.fasting` | bool | ناشتا بودم | route `/labs/new` | — | — |
| `labu.calendar` | date | روز آزمایش | route `/labs/new` | yes | — |
| `labu.date-done` | action | تأیید روز | route `/labs/new` | yes | — |
| `labv.marker:*` | action | — | — | — | — |
| `labv.confirm:*` | action | — | — | — | — |
| `labv.confirm-all` | action | تأیید همه‌ی مقادیر | — | — | — |
| `labv.add` | action | افزودن مقدار | — | — | — |
| `labv.field:name` | text | نام شاخص | — | yes | — |
| `labv.field:value` | text | مقدار شاخص | — | yes | — |
| `labv.field:unit` | text | واحد شاخص | — | yes | — |
| `labv.field:ref_low` | text | حداقل مرجع | — | yes | — |
| `labv.field:ref_high` | text | حداکثر مرجع | — | yes | — |
| `labv.save` | action | ذخیره‌ی مقدار | — | yes | — |
| `labr.share` | action | ارسال نتیجه برای دکتر | — | — | — |
| `labr.flag:*` | action | — | — | — | — |
| `labr.confirm-values` | action | تأیید مقادیر | — | — | — |
| `labr.edit-values` | action | ویرایش مقادیر | — | — | — |
| `labr.all-normal` | section | شاخص‌های طبیعی | — | — | — |
| `labr.marker:*` | action | — | — | — | — |
| `labr.feedback` | single | نظر درباره‌ی تحلیل | — | — | `true` `false` |
| `hrec.report` | action | گزارش برای دکتر | route `/record` | — | — |
| `hrec.edit:basics` | action | ویرایش مشخصات پایه | route `/record` | — | — |
| `hrec.edit:conditions` | action | ویرایش بیماری‌ها | route `/record` | — | — |
| `hrec.edit:allergies` | action | ویرایش حساسیت‌ها | route `/record` | — | — |
| `hrec.edit:pregnancies` | action | ویرایش سوابق بارداری | route `/record` | — | — |
| `hrec.open:medications` | action | مدیریت داروها | route `/record` | — | — |
| `hrec.open:vitals` | action | علائم حیاتی | route `/record` | — | — |
| `hrec.checkup:*` | action | — | route `/record` | — | — |
| `hrec.lab:*` | action | — | route `/record` | — | — |
| `hrec.blood` | single | گروه خونی | route `/record` | yes | `A+` `A-` `B+` `B-` `AB+` `AB-` `O+` `O-` `unknown` |
| `hrec.edit-height` | action | ویرایش قد | route `/record` | yes | — |
| `hrec.edit-weight` | action | ویرایش وزن | route `/record` | yes | — |
| `hrec.basics-save` | action | ذخیره‌ی مشخصات پایه | route `/record` | yes | — |
| `hrec.chronic` | multi | بیماری‌های زمینه‌ای | route `/record` | yes | `diabetes` `hypertension` `thyroid` `asthma` `anemia` `migraine` `other` |
| `hrec.gyn` | multi | مشکلات زنان | route `/record` | yes | `pcos` `endometriosis` `fibroids` `recurrent_infections` `other` |
| `hrec.conditions-save` | action | ذخیره‌ی بیماری‌ها | route `/record` | yes | — |
| `hrec.allergy-input` | text | نام حساسیت | route `/record` | yes | — |
| `hrec.allergy-add` | action | افزودن حساسیت | route `/record` | yes | — |
| `hrec.allergy-none` | bool | حساسیتی ندارم | route `/record` | yes | — |
| `hrec.allergies-save` | action | ذخیره‌ی حساسیت‌ها | route `/record` | yes | — |
| `hrec.preg-add` | action | افزودن سابقه‌ی بارداری | route `/record` | yes | — |
| `hrec.preg-entry:*` | action | — | route `/record` | yes | — |
| `hrec.preg-outcome` | single | پایان بارداری | route `/record` | yes | `vaginal` `cesarean` `ended` |
| `hrec.preg-year` | single | سال بارداری | route `/record` | yes | — |
| `hrec.preg-babies` | integer | تعداد نوزاد | route `/record` | yes | — |
| `hrec.preg-save` | action | ذخیره‌ی سابقه‌ی بارداری | route `/record` | yes | — |
| `rec.allergy-card` | bool | حساسیت دارویی روی کارت اورژانسی | route `/record` | — | — |
| `rec.tile:imaging` | action | تصویربرداری | route `/record` | — | — |
| `rec.tile:visit` | action | ویزیت | route `/record` | — | — |
| `rec.tile:labs` | action | آزمایش‌ها | route `/record` | — | — |
| `rec.tile:prescription` | action | نسخه‌ها | route `/record` | — | — |
| `rec.tile:hospital` | action | بستری | route `/record` | — | — |
| `rec.tile:other` | action | سایر اسناد | route `/record` | — | — |
| `rec.tile:surgeries` | action | جراحی‌ها | route `/record` | — | — |
| `rec.tile:family_history` | action | سابقه‌ی خانوادگی | route `/record` | — | — |
| `rec.tile:all` | action | همه‌ی اسناد | route `/record` | — | — |
| `rec.review` | action | اسناد نیازمند بازبینی | route `/record` | — | — |
| `rec.all-docs` | action | همه‌ی اسناد | route `/record` | — | — |
| `rec.add-doc` | action | افزودن سند | route `/record` | — | — |
| `rec.emergency` | action | کارت اورژانسی | route `/record` | — | — |
| `rec.share` | action | اشتراک پرونده | route `/record` | — | — |
| `rec.surgery-input` | text | نام جراحی | route `/record` | yes | — |
| `rec.surgery-add` | action | افزودن جراحی | route `/record` | yes | — |
| `rec.surgery-date` | action | تاریخ جراحی | route `/record` | yes | — |
| `rec.surgery-date-cal` | date | روز جراحی | route `/record` | yes | — |
| `rec.surgery-date-clear` | action | بدون تاریخ جراحی | route `/record` | yes | — |
| `rec.family-input` | text | بیماری در خانواده | route `/record` | yes | — |
| `rec.family-add` | action | افزودن سابقه‌ی خانوادگی | route `/record` | yes | — |
| `rec.relative` | single | نسبت فامیلی | route `/record` | yes | — |
| `rec.extras-save` | action | ذخیره‌ی سوابق | route `/record` | yes | — |
| `rec.doc-kind` | single | نوع سند | route `/record` | yes | `lab` `imaging` `visit` `prescription` `hospital` `other` |
| `rec.doc-title` | text | عنوان سند | route `/record` | yes | — |
| `rec.doc-date` | action | تاریخ سند | route `/record` | yes | — |
| `rec.doc-date-cal` | date | روز سند | route `/record` | yes | — |
| `rec.doc-date-clear` | action | بدون تاریخ سند | route `/record` | yes | — |
| `recd.edit` | action | ویرایش سند | — | — | — |
| `recd.manual` | action | وارد کردن دستی | — | — | — |
| `recd.download` | action | دانلود سند | — | — | — |
| `recd.dating-apply` | action | اعمال تاریخ پیشنهادی زایمان | — | — | — |
| `recd.dating-dismiss` | action | رد تاریخ پیشنهادی | — | — | — |
| `recd.kind` | single | نوع سند | — | yes | `lab` `imaging` `visit` `prescription` `hospital` `other` |
| `recd.title` | text | عنوان سند | — | yes | — |
| `recd.centre` | text | مرکز درمانی | — | yes | — |
| `recd.doctor` | text | نام دکتر | — | yes | — |
| `recd.note` | text | یادداشت سند | — | yes | — |
| `recd.date` | action | تاریخ سند | — | yes | — |
| `recd.ended-on` | action | تاریخ ترخیص | — | yes | — |
| `recd.save` | action | ذخیره‌ی سند | — | yes | — |
| `recd.date-cal` | date | روز سند | — | yes | — |
| `recd.date-clear` | action | بدون تاریخ | — | yes | — |
| `recd.ended-on-cal` | date | روز ترخیص | — | yes | — |
| `recd.ended-on-clear` | action | بدون تاریخ ترخیص | — | yes | — |
| `recd.study` | single | نوع تصویربرداری | — | — | `ultrasound` `xray` `mri` `ct` `mammography` `bone_density` `other` |
| `recd.read:*` | text | — | — | — | — |
| `recd.readnum:*` | number | — | — | — | — |
| `recd.readdate:*` | action | — | — | — | — |
| `recd.readdate-cal:*` | date | — | — | — | — |
| `recd.readdate-clear:*` | action | — | — | — | — |
| `recd.item:*` | text | — | — | — | — |
| `recd.item-add` | action | افزودن قلم نسخه | — | — | — |
| `recd.read-confirm` | action | تأیید اطلاعات سند | — | — | — |
| `recem.lock` | bool | نمایش روی صفحه‌ی قفل | route `/record/emergency` | — | — |
| `recem.pregnancy` | bool | نمایش بارداری روی کارت | route `/record/emergency` | — | — |
| `recem.allergies` | bool | نمایش حساسیت‌ها روی کارت | route `/record/emergency` | — | — |
| `recem.contact` | action | مخاطب اورژانسی | route `/record/emergency` | — | — |
| `recem.insurance` | action | بیمه | route `/record/emergency` | — | — |
| `recem.contact-name` | text | نام مخاطب اورژانسی | route `/record/emergency` | yes | — |
| `recem.contact-relation` | text | نسبت با مخاطب | route `/record/emergency` | yes | — |
| `recem.contact-phone` | text | شماره‌ی مخاطب اورژانسی | route `/record/emergency` | yes | — |
| `recem.contact-save` | action | ذخیره‌ی مخاطب اورژانسی | route `/record/emergency` | yes | — |
| `recem.insurance-label` | text | نام بیمه | route `/record/emergency` | yes | — |
| `recem.insurance-last4` | text | ۴ رقم آخر بیمه | route `/record/emergency` | yes | — |
| `recem.insurance-save` | action | ذخیره‌ی بیمه | route `/record/emergency` | yes | — |
| `recx.preview` | action | پیش‌نمایش گزارش | route `/record/export` | — | — |
| `recx.range` | single | بازه‌ی گزارش | route `/record/export` | — | `3m` `6m` `1y` `custom` |
| `recx.custom-from` | action | تاریخ شروع دلخواه | route `/record/export` | — | — |
| `recx.group:basics` | bool | بخش مشخصات پایه در گزارش | route `/record/export` | — | — |
| `recx.group:health` | bool | بخش سلامت در گزارش | route `/record/export` | — | — |
| `recx.group:cycle` | bool | بخش چرخه در گزارش | route `/record/export` | — | — |
| `recx.group:vitals` | bool | بخش علائم حیاتی در گزارش | route `/record/export` | — | — |
| `recx.group:checkups` | bool | بخش چکاپ و آزمایش در گزارش | route `/record/export` | — | — |
| `recx.group:pregnancies` | bool | بخش بارداری در گزارش | route `/record/export` | — | — |
| `recx.losses` | bool | شامل سقط و از دست دادن | route `/record/export` | yes | — |
| `recx.question` | text | سؤال برای دکتر | route `/record/export` | — | — |
| `recx.download` | action | دانلود گزارش | route `/record/export` | — | — |
| `recx.custom-cal` | date | روز شروع گزارش | route `/record/export` | — | — |
| `recx.custom-done` | action | تأیید روز شروع | route `/record/export` | — | — |
| `recx.preview-download` | action | دانلود گزارش | route `/record/export/preview` | — | — |
| `mrp.range` | single | بازه‌ی گزارش یائسگی | route `/menopause/report` | — | `1` `3` `6` |
| `mrp.question` | text | سؤال برای دکتر | route `/menopause/report` | — | — |
| `mrp.download` | action | دانلود گزارش یائسگی | route `/menopause/report` | — | — |
| `rectl.filter` | single | نوع اسناد | route `/record/timeline` | — | `all` `lab` `imaging` `visit` `prescription` `hospital` `other` |
| `rectl.add` | action | افزودن سند | route `/record/timeline` | — | — |
| `rectl.add-header` | action | افزودن سند | route `/record/timeline` | — | — |
| `rectl.more` | action | اسناد بیشتر | route `/record/timeline` | — | — |
| `rectl.item:*` | action | — | route `/record/timeline` | — | — |
| `backup.json` | action | خروجی JSON داده‌ها | route `/profile/backup` | — | — |
| `backup.pdf` | action | خروجی PDF | route `/profile/backup` | — | — |
| `backup.report` | action | گزارش برای دکتر | route `/profile/backup` | — | — |
| `devices.notify:watch` | bool | خبرم کن، ساعت هوشمند | route `/profile/devices` | — | — |
| `devices.notify:band` | bool | خبرم کن، مچ‌بند | route `/profile/devices` | — | — |
| `devices.notify:scale` | bool | خبرم کن، ترازو | route `/profile/devices` | — | — |
| `devices.notify:thermometer` | bool | خبرم کن، دماسنج | route `/profile/devices` | — | — |
| `devices.notify:pressure` | bool | خبرم کن، فشارسنج | route `/profile/devices` | — | — |
| `devices.notify:glucose` | bool | خبرم کن، قندسنج | route `/profile/devices` | — | — |
| `devices.vitals` | action | ثبت دستی علائم حیاتی | route `/profile/devices` | — | — |
| `menopause.alert-book` | action | وقت پزشک زنان | route `/menopause/alert` | — | — |
| `menopause.alert-report` | action | گزارش برای پزشک | route `/menopause/alert` | — | — |
| `menopause.stage` | single | مرحله‌ی یائسگی | route `/menopause/stage` | — | `peri` `meno` `post` `unsure` |
| `menopause.stage-month-prev` | action | ماه قبل (آخرین پریود) | route `/menopause/stage` | — | — |
| `menopause.stage-month-next` | action | ماه بعد (آخرین پریود) | route `/menopause/stage` | — | — |
| `menopause.stage-surgical` | single | یائسگی جراحی | route `/menopause/stage` | — | `yes` `no` |
| `menopause.stage-hrt` | single | درمان هورمونی | route `/menopause/stage` | — | `yes` `no` |
| `menopause.stage-save` | action | ذخیره‌ی مرحله | route `/menopause/stage` | — | — |
| `menopause.flash-toggle` | action | شروع/پایان گرگرفتگی | route `/menopause/hot-flash` | — | — |
| `menopause.flash-severity` | single | شدت گرگرفتگی | route `/menopause/hot-flash` | — | `mild` `moderate` `severe` `very_severe` |
| `menopause.flash-sweat` | multi | عرق کردن | route `/menopause/hot-flash` | — | `sweat` |
| `menopause.flash-triggers` | multi | محرک‌های گرگرفتگی | route `/menopause/hot-flash` | — | `caffeine` `stress` `spicy_food` `warm_room` `hot_drink` `exercise` `unknown` |
| `menopause.score-fill` | action | پر کردن پرسش‌نامه‌ی ماه | route `/menopause/score` | — | — |
| `menopause.score-analysis` | action | تحلیل علائم | route `/menopause/score` | — | — |
| `menopause.score-q:*` | integer | — | route `/menopause/score/questionnaire` | — | — |
| `menopause.score-submit` | action | ثبت پرسش‌نامه | route `/menopause/score/questionnaire` | — | — |
| `menopause.treat-edit:*` | action | — | route `/menopause/treatment` | — | — |
| `menopause.treat-take:*` | bool | — | route `/menopause/treatment` | — | — |
| `menopause.treat-goal:*` | action | — | route `/menopause/treatment` | — | — |
| `menopause.treat-suggest:*` | action | — | route `/menopause/treatment` | — | — |
| `menopause.treat-side` | multi | عوارض امروز | route `/menopause/treatment` | — | `breast_tenderness` `spotting` `headache` `bloating` `mood_change` |
| `menopause.treat-name` | text | نام دارو یا برنامه | — | — | — |
| `menopause.treat-dose` | text | دوز | — | — | — |
| `menopause.treat-schedule` | single | زمان مصرف | — | — | `morning` `noon` `evening` `night` `weekly` |
| `menopause.treat-remind` | bool | یادآور مصرف | — | — | — |
| `menopause.treat-unit` | single | واحد هدف | — | — | `sessions` `minutes` |
| `menopause.treat-goal` | integer | هدف هفتگی | — | — | — |
| `menopause.treat-started` | action | تاریخ شروع | — | — | — |
| `menopause.treat-review` | action | تاریخ مرور با پزشک | — | — | — |
| `menopause.treat-date` | date | تاریخ | — | — | — |
| `menopause.treat-date-done` | action | تایید تاریخ | — | — | — |
| `menopause.treat-save` | action | ذخیره | — | — | — |
| `menopause.treat-stop` | action | قطع مصرف | — | — | — |
| `menopause.treat-resume` | action | از سرگیری | — | — | — |
| `menopause.treat-minutes` | integer | دقیقه‌ی امروز | — | — | — |
| `menopause.treat-minutes-save` | action | ثبت دقیقه‌ها | — | — | — |
| `programs.card:*` | action | — | route `/programs` | — | — |
| `programs.pelvic-card` | action | برنامه‌ی کف لگن | route `/programs` | — | — |
| `programs.join:*` | action | — | — | — | — |
| `programs.open:*` | action | — | — | — | — |
| `programs.bleeding-join` | action | پیوستن به پیگیری خونریزی | route `/programs/bleeding` | — | — |
| `programs.pbac-pads:*` | integer | — | route `/programs/bleeding` | — | — |
| `programs.pbac-clots` | single | لخته | route `/programs/bleeding` | — | `none` `small` `large` |
| `programs.pbac-flooding` | bool | خونریزی ناگهانی شدید | route `/programs/bleeding` | — | — |
| `programs.pbac-save` | action | ذخیره‌ی نمودار خونریزی | route `/programs/bleeding` | — | — |
| `programs.pain-join` | action | پیوستن به دفتر درد | route `/programs/pain` | — | — |
| `programs.pain-locations` | multi | محل درد | route `/programs/pain` | — | — |
| `programs.pain-score` | integer | شدت درد | route `/programs/pain` | — | — |
| `programs.pain-types` | multi | نوع درد | route `/programs/pain` | — | — |
| `programs.pain-with` | multi | علائم همراه درد | route `/programs/pain` | — | — |
| `programs.pain-missed` | bool | از فعالیت جا موندم | route `/programs/pain` | — | — |
| `programs.pain-analgesic` | text | مسکن مصرفی | route `/programs/pain` | — | — |
| `programs.pain-analgesic-time` | time | ساعت مسکن | route `/programs/pain` | — | — |
| `programs.pain-effect` | single | اثر مسکن | route `/programs/pain` | — | `no` `a_little` `helped` |
| `programs.pain-save` | action | ذخیره‌ی دفتر درد | route `/programs/pain` | — | — |
| `programs.pmdd-join` | action | پیوستن به پیگیری PMDD | route `/programs/pmdd` | — | — |
| `programs.pmdd-q:*` | integer | — | route `/programs/pmdd` | — | — |
| `programs.pmdd-save` | action | ذخیره‌ی پرسش‌نامه‌ی PMDD | route `/programs/pmdd` | — | — |
| `pelvic.session-start` | action | شروع تمرین کگل | route `/programs/pelvic` | — | — |
| `pelvic.program-join` | action | شروع برنامه‌ی کف لگن | route `/programs/pelvic` | — | — |
| `pelvic.leak` | single | نشت ادرار | route `/programs/pelvic` | — | `none` `cough` `urgency` `unexplained` |
| `pelvic.night-voids` | integer | دفعات بیدار شدن شبانه برای دستشویی | route `/programs/pelvic` | — | — |
| `pelvic.uti-symptoms` | multi | علائم عفونت ادراری | route `/programs/pelvic` | — | `burning` `frequency` `cloudy_odor` |
| `pelvic.kegel-begin` | action | شروع کگل | route `/programs/pelvic/kegel` | — | — |
| `pelvic.kegel-stop` | action | توقف کگل | route `/programs/pelvic/kegel` | — | — |
| `pelvic.kegel-done` | action | برگشت به برنامه | route `/programs/pelvic/kegel` | — | — |
| `contraception.setup-open` | action | انتخاب روش پیشگیری | route `/contraception` | — | — |
| `contraception.settings` | action | تنظیمات پیشگیری | route `/contraception` | — | — |
| `contraception.pill-taken` | action | قرصم رو خوردم | route `/contraception` | — | — |
| `contraception.pill-undo` | action | برگردوندن قرص امروز | route `/contraception` | — | — |
| `contraception.missed-open` | action | قرصم یادم رفت | route `/contraception` | — | — |
| `contraception.packs-edit` | action | تعداد بسته‌های باقی‌مانده | route `/contraception` | — | — |
| `contraception.packs-left` | integer | بسته‌های باقی‌مانده | — | — | — |
| `contraception.packs-save` | action | ذخیره‌ی بسته‌ها | — | — | — |
| `contraception.method-change` | action | تغییر روش | route `/contraception` | — | — |
| `contraception.reminders-open` | action | یادآورهای روش | route `/contraception` | — | — |
| `contraception.missed-count` | single | تعداد قرص‌های جاافتاده | route `/contraception/missed` | — | — |
| `contraception.missed-back` | action | برگشت به پیشگیری | route `/contraception/missed` | — | — |
| `contraception.iud-set` | action | تاریخ گذاشتن IUD | route `/contraception/other` | — | — |
| `contraception.iud-string` | bool | چک نخ IUD | route `/contraception/other` | — | — |
| `contraception.iud-followup-done` | action | ثبت ویزیت پیگیری IUD | route `/contraception/other` | — | — |
| `contraception.injection-set` | action | تاریخ تزریق | route `/contraception/other` | — | — |
| `contraception.injection-done` | action | تزریق انجام شد | route `/contraception/other` | — | — |
| `contraception.implant-date` | action | تاریخ تعویض ایمپلنت | route `/contraception/other` | — | — |
| `contraception.date` | date | تاریخ | — | — | — |
| `contraception.date-save` | action | ذخیره‌ی تاریخ | — | — | — |
| `contraception.method` | single | روش پیشگیری | route `/contraception/setup` | — | `combined_pill` `progestin_pill` `copper_iud` `hormonal_iud` `injection` `implant` `condom` `other` |
| `contraception.pack-type` | single | نوع بسته‌ی قرص | route `/contraception/setup` | — | `21_7` `28` `24_4` |
| `contraception.pack-start` | action | شروع بسته | route `/contraception/setup` | — | — |
| `contraception.reminder-time` | action | ساعت یادآور قرص | route `/contraception/setup` | — | — |
| `contraception.iud-inserted` | action | تاریخ گذاشتن IUD | route `/contraception/setup` | — | — |
| `contraception.iud-years` | integer | عمر IUD | route `/contraception/setup` | — | — |
| `contraception.injected-on` | action | تاریخ آخرین تزریق | route `/contraception/setup` | — | — |
| `contraception.implant-inserted` | action | تاریخ گذاشتن ایمپلنت | route `/contraception/setup` | — | — |
| `contraception.setup-save` | action | ذخیره‌ی روش | route `/contraception/setup` | — | — |
| `contraception.setup-date` | date | تاریخ | — | — | — |
| `contraception.time-hour` | single | ساعت | — | — | — |
| `contraception.time-minute` | single | دقیقه | — | — | — |
| `contraception.picker-done` | action | تایید | — | — | — |
| `ivf.off-cta` | action | فعال کردن IVF | route `/ivf` | — | — |
| `ivf.cycle-start` | action | شروع سیکل IVF | — | — | — |
| `ivf.cycle-edit` | action | ویرایش سیکل | route `/ivf` | — | — |
| `ivf.dose:*` | bool | — | — | — | — |
| `ivf.next-appointment` | action | نوبت بعدی | route `/ivf` | — | — |
| `ivf.companion-notify` | bool | خبر دادن به همراه | route `/ivf` | — | — |
| `ivf.action:*` | action | — | route `/ivf` | — | — |
| `ivf.setup-open` | action | باز کردن سیکل فعلی | route `/ivf/cycle/new` | — | — |
| `ivf.setup-protocol` | single | پروتکل | route `/ivf/cycle/new` | — | — |
| `ivf.setup-stage` | single | الان کجای مسیری | route `/ivf/cycle/new` | — | `prep` `stim` `retrieval` `transfer` `tww` `test` |
| `ivf.setup-started` | action | تاریخ شروع سیکل | route `/ivf/cycle/new` | — | — |
| `ivf.setup-stim-started` | action | تاریخ شروع تحریک | route `/ivf/cycle/new` | — | — |
| `ivf.setup-submit` | action | شروع سیکل | route `/ivf/cycle/new` | — | — |
| `ivf.cycle-stage` | single | مرحله‌ی سیکل | route `/ivf/cycle` | — | `prep` `stim` `retrieval` `transfer` `tww` `test` |
| `ivf.cycle-protocol` | single | پروتکل | route `/ivf/cycle` | — | — |
| `ivf.cycle-started` | action | تاریخ شروع سیکل | route `/ivf/cycle` | — | — |
| `ivf.cycle-stim-started` | action | تاریخ شروع تحریک | route `/ivf/cycle` | — | — |
| `ivf.cycle-beta` | action | تاریخ تست بتا | route `/ivf/cycle` | — | — |
| `ivf.cycle-nextscan-date` | action | تاریخ سونوی بعدی | route `/ivf/cycle` | — | — |
| `ivf.cycle-nextscan-time` | action | ساعت سونوی بعدی | route `/ivf/cycle` | — | — |
| `ivf.cycle-retrieval-date` | action | تاریخ جمع‌آوری تخمک | route `/ivf/cycle` | — | — |
| `ivf.cycle-retrieval-time` | action | ساعت جمع‌آوری تخمک | route `/ivf/cycle` | — | — |
| `ivf.cycle-transfer-date` | action | تاریخ انتقال | route `/ivf/cycle` | — | — |
| `ivf.cycle-transfer-time` | action | ساعت انتقال | route `/ivf/cycle` | — | — |
| `ivf.cycle-save` | action | ذخیره‌ی سیکل | route `/ivf/cycle` | — | — |
| `ivf.cycle-stop-keep` | action | نگه داشتن یادآورها | — | — | — |
| `ivf.cycle-stop-confirm` | action | قطع یادآورهای دارو | — | — | — |
| `ivf.date` | date | تاریخ | — | — | — |
| `ivf.time-hour` | single | ساعت | — | — | — |
| `ivf.time-minute` | single | دقیقه | — | — | — |
| `ivf.picker-done` | action | تایید | — | — | — |
| `ivf.meds-home` | action | برگشت به IVF | route `/ivf/meds` | — | — |
| `ivf.med-add` | action | افزودن دارو | route `/ivf/meds` | — | — |
| `ivf.site` | single | محل تزریق | route `/ivf/meds` | — | — |
| `ivf.stock:*` | action | — | route `/ivf/meds` | — | — |
| `ivf.med-preset` | single | داروی آماده | route `/ivf/meds/new` | — | — |
| `ivf.med-name` | text | نام دارو | route `/ivf/meds/new` | — | — |
| `ivf.med-role` | single | نقش دارو | route `/ivf/meds/new` | — | `stimulation` `suppression` `trigger` `luteal_support` `other` |
| `ivf.med-route` | single | راه مصرف | route `/ivf/meds/new` | — | `subcutaneous` `intramuscular` `oral` `vaginal` `other` |
| `ivf.med-dose` | text | دوز | route `/ivf/meds/new` | — | — |
| `ivf.med-unit` | single | واحد دوز | route `/ivf/meds/new` | — | `iu` `mg` `mcg` `ml` |
| `ivf.med-trigger-date` | action | تاریخ تریگر | route `/ivf/meds/new` | — | — |
| `ivf.med-trigger-time` | action | ساعت تریگر | route `/ivf/meds/new` | — | — |
| `ivf.med-time:*` | action | — | route `/ivf/meds/new` | — | — |
| `ivf.med-time-add` | action | افزودن ساعت مصرف | route `/ivf/meds/new` | — | — |
| `ivf.med-starts` | action | تاریخ شروع دارو | route `/ivf/meds/new` | — | — |
| `ivf.med-ends` | action | تاریخ پایان دارو | route `/ivf/meds/new` | — | — |
| `ivf.med-track-stock` | bool | پیگیری موجودی | route `/ivf/meds/new` | — | — |
| `ivf.med-stock-unit` | single | واحد موجودی | route `/ivf/meds/new` | — | `pen` `vial` `ampoule` `prefilled_syringe` `box` `other` |
| `ivf.med-stock` | integer | موجودی | route `/ivf/meds/new` | — | — |
| `ivf.med-doses-per-unit` | integer | تعداد دوز در هر واحد | route `/ivf/meds/new` | — | — |
| `ivf.med-notes` | text | یادداشت دارو | route `/ivf/meds/new` | — | — |
| `ivf.med-active` | bool | فعال بودن یادآور دارو | — | — | — |
| `ivf.med-save` | action | ذخیره‌ی دارو | route `/ivf/meds/new` | — | — |
| `ivf.scan-home` | action | برگشت به IVF | route `/ivf/scan` | — | — |
| `ivf.scan-follicles:*` | integer | — | route `/ivf/scan` | — | — |
| `ivf.scan-endometrium` | text | ضخامت آندومتر | route `/ivf/scan` | — | — |
| `ivf.scan-e2` | text | استرادیول | route `/ivf/scan` | — | — |
| `ivf.scan-e2-unit` | action | واحد استرادیول | route `/ivf/scan` | — | — |
| `ivf.scan-save` | action | ذخیره‌ی سونو | route `/ivf/scan` | — | — |
| `ivf.tww-home` | action | برگشت به IVF | route `/ivf/tww` | — | — |
| `ivf.tww-mood` | single | حال امروز | route `/ivf/tww` | — | `calm` `hopeful` `worried` `tired` |
| `ivf.tww-symptoms` | action | ثبت علائم | route `/ivf/tww` | — | — |
| `ivf.tww-med-add` | action | افزودن داروی حمایتی | route `/ivf/tww` | — | — |
| `ivf.tww-result:*` | action | — | route `/ivf/tww` | — | — |
| `ivf.tww-confirm` | action | تایید نتیجه | — | — | — |
| `ivf.tww-next:*` | action | — | route `/ivf/tww` | — | — |
| `mode.life` | single | حالت زندگی | route `/profile/mode` | — | `cycle` `ttc` `pregnancy` `postpartum` `menopause` `teen` |
| `mode.ivf-iui` | bool | IVF / IUI | route `/profile/mode` | — | — |
| `mode.loss-open` | action | مراقبت بعد از از دست دادن | route `/profile/mode` | — | — |
| `mode.contraception` | bool | پیگیری پیشگیری | route `/profile/mode` | — | — |
| `mode.contraception-manage` | action | مدیریت پیشگیری | route `/profile/mode` | — | — |
| `mode.confirm-go` | action | تایید تغییر حالت | — | — | — |
| `me.open:profile-appearance` | action | باز کردن «ظاهر اپ» | route `/profile` | — | — |
| `me.open:profile-account` | action | باز کردن «حساب من» | route `/profile` | — | — |
| `me.open:profile-credits` | action | باز کردن «اعتبار هوش مصنوعی» | route `/profile` | — | — |
| `me.open:companion-links` | action | باز کردن «کد همدم» | route `/profile` | — | — |
| `me.open:profile-mode` | action | باز کردن «مرحله‌ی زندگی» | route `/profile` | — | — |
| `me.open:companions` | action | باز کردن «همدم‌ها و خانواده» | route `/profile` | — | — |
| `me.open:children` | action | باز کردن «فرزندان» | route `/profile` | — | — |
| `me.open:todo` | action | باز کردن «کارهای من» | route `/profile` | — | — |
| `me.open:learn` | action | باز کردن «دوره‌های من» | route `/profile` | — | — |
| `me.open:learn-downloads` | action | باز کردن «دانلودها» | route `/profile` | — | — |
| `me.open:profile-bookings` | action | باز کردن «رزروها» | route `/profile` | — | — |
| `me.open:profile-orders` | action | باز کردن «سفارش‌ها» | route `/profile` | — | — |
| `me.open:services` | action | باز کردن «خدمات» | route `/profile` | — | — |
| `me.open:vitals` | action | باز کردن «علائم حیاتی» | route `/profile` | — | — |
| `me.open:record` | action | باز کردن «پرونده‌ی سلامت» | route `/profile` | — | — |
| `me.open:profile-devices` | action | باز کردن «دستگاه‌ها» | route `/profile` | — | — |
| `me.open:profile-backup` | action | باز کردن «پشتیبان‌گیری» | route `/profile` | — | — |
| `me.open:profile-privacy` | action | باز کردن «حریم خصوصی» | route `/profile` | — | — |
| `me.open:profile-notifications` | action | باز کردن «اعلان‌ها» | route `/profile` | — | — |
| `me.open:cycle-settings` | action | باز کردن «تنظیمات چرخه» | route `/profile` | — | — |
| `me.open:profile-language` | action | باز کردن «زبان و تقویم» | route `/profile` | — | — |
| `me.open:profile-support` | action | باز کردن «پشتیبانی» | route `/profile` | — | — |
| `me.open:profile-support-chat` | action | باز کردن «گفت‌وگو با پشتیبانی» | route `/profile` | — | — |
| `me.open:profile-about` | action | باز کردن «درباره‌ی ریتمی» | route `/profile` | — | — |
| `me.open:plus` | action | باز کردن «اشتراک ریتمی پلاس» | route `/profile` | — | — |
| `me.header:appearance` | action | ظاهر اپ (بالای صفحه) | route `/profile` | — | — |
| `account.edit:name` | action | ویرایش نام | route `/profile/account` | — | — |
| `account.edit:birthday` | action | ویرایش تاریخ تولد | route `/profile/account` | — | — |
| `account.edit:cycleDuration` | action | ویرایش طول چرخه | route `/profile/account` | — | — |
| `account.edit:periodDuration` | action | ویرایش طول پریود | route `/profile/account` | — | — |
| `account.edit:lastPeriod` | action | ویرایش شروع آخرین پریود | route `/profile/account` | — | — |
| `account.edit:weight` | action | ویرایش وزن | route `/profile/account` | — | — |
| `account.edit:height` | action | ویرایش قد | route `/profile/account` | — | — |
| `quick.name` | text | نام | — | yes | — |
| `quick.cycle-length` | integer | طول چرخه | — | yes | — |
| `quick.period-length` | integer | طول پریود | — | yes | — |
| `quick.birthday:day` | integer | روز تولد | — | yes | — |
| `quick.birthday:month` | integer | ماه تولد | — | yes | — |
| `quick.birthday:year` | integer | سال تولد | — | yes | — |
| `quick.last-period:day` | integer | روز آخرین پریود | — | yes | — |
| `quick.last-period:month` | integer | ماه آخرین پریود | — | yes | — |
| `quick.last-period:year` | integer | سال آخرین پریود | — | yes | — |
| `quick.weight` | number | وزن | — | yes | — |
| `quick.height` | number | قد | — | yes | — |
| `quick.save` | action | ذخیره‌ی ویرایش | — | — | — |
| `health.cycle-length` | integer | طول چرخه | sheet `health` | — | — |
| `health.period-length` | integer | طول پریود | sheet `health` | — | — |
| `health.last-period:day` | integer | روز آخرین پریود | sheet `health` | — | — |
| `health.last-period:month` | integer | ماه آخرین پریود | sheet `health` | — | — |
| `health.last-period:year` | integer | سال آخرین پریود | sheet `health` | — | — |
| `health.save` | action | ذخیره‌ی اطلاعات سلامت | sheet `health` | — | — |
| `personal.name` | text | نام | sheet `personal` | — | — |
| `personal.birthday:day` | integer | روز تولد | sheet `personal` | — | — |
| `personal.birthday:month` | integer | ماه تولد | sheet `personal` | — | — |
| `personal.birthday:year` | integer | سال تولد | sheet `personal` | — | — |
| `personal.weight` | number | وزن | sheet `personal` | — | — |
| `personal.height` | number | قد | sheet `personal` | — | — |
| `personal.save` | action | ذخیره‌ی اطلاعات شخصی | sheet `personal` | — | — |
| `language.pick` | single | زبان اپ | — | — | — |
| `reminders.add` | action | یادآور جدید | sheet `reminders` | — | — |
| `reminders.add-first` | action | اولین یادآور | sheet `reminders` | — | — |
| `reminders.type` | single | نوع یادآور | sheet `reminders` | yes | — |
| `reminders.title` | text | عنوان یادآور | sheet `reminders` | yes | — |
| `reminders.subtitle` | text | توضیح یادآور | sheet `reminders` | yes | — |
| `reminders.recurrence` | single | تکرار یادآور | sheet `reminders` | yes | — |
| `reminders.time` | time | ساعت یادآور | sheet `reminders` | yes | — |
| `reminders.submit` | action | ذخیره‌ی یادآور | sheet `reminders` | yes | — |
| `reminders.toggle:*` | bool | — | sheet `reminders` | — | — |
| `notifications.history` | action | سابقه‌ی هشدارها | sheet `notifications` | — | — |
| `notifications.mark-all` | action | همه خوانده شد | sheet `notifications` | — | — |
| `appearance.theme` | single | تم اپ | route `/profile/appearance` | — | `light` `dark` `system` |
| `appearance.follow-system` | bool | پیروی از تم سیستم | route `/profile/appearance` | — | — |
| `appearance.text-size` | integer | اندازه‌ی نوشته | route `/profile/appearance` | — | — |
| `appearance.reduce-motion` | bool | کاهش حرکت | route `/profile/appearance` | — | — |
| `notif.category:*` | bool | — | route `/profile/notifications` | — | — |
| `notif.quiet` | bool | ساعت سکوت | route `/profile/notifications` | — | — |
| `notif.quiet-edit` | action | ویرایش بازه‌ی سکوت | route `/profile/notifications` | — | — |
| `notif.quiet-start` | time | شروع ساعت سکوت | route `/profile/notifications` | yes | — |
| `notif.quiet-end` | time | پایان ساعت سکوت | route `/profile/notifications` | yes | — |
| `notif.quiet-done` | action | تأیید بازه‌ی سکوت | route `/profile/notifications` | yes | — |
| `notif.sms` | bool | پیامک یادآور | route `/profile/notifications` | — | — |
| `notif.neutral-copy` | bool | متن خنثی اعلان‌ها | route `/profile/notifications` | — | — |
| `privacy.open:companions` | action | همدم‌ها | route `/profile/privacy` | — | — |
| `privacy.open:companions-footer` | action | مدیریت همدم‌ها | route `/profile/privacy` | — | — |
| `privacy.discreet` | bool | اعلان‌های محرمانه | route `/profile/privacy` | — | — |
| `privacy.consent:*` | bool | — | route `/profile/privacy` | — | — |
| `privacy.export` | action | دریافت نسخه‌ی داده‌ها | route `/profile/privacy` | — | — |
| `privacy.export-json` | action | خروجی JSON | route `/profile/privacy` | yes | — |
| `privacy.export-pdf` | action | خروجی PDF | route `/profile/privacy` | yes | — |
| `about.open:terms` | action | شرایط استفاده | route `/profile/about` | — | — |
| `about.open:privacy` | action | سیاست حریم خصوصی | route `/profile/about` | — | — |
| `legal.tab:*` | action | — | route `/profile/legal` | — | — |
| `support.search` | text | جست‌وجو در پرسش‌های متداول | route `/profile/support` | — | — |
| `support.faq:*` | section | — | route `/profile/support` | — | — |
| `support.report` | action | گزارش مشکل | route `/profile/support` | — | — |
| `report.message` | text | متن گزارش مشکل | — | yes | — |
| `report.send` | action | ارسال گزارش | — | yes | — |
| `plus.open:plus-manage` | action | مدیریت اشتراک | route `/plus` | — | — |
| `plus.open:plus-plans` | action | دیدن پلن‌ها | route `/plus` | — | — |
| `plus.plan` | single | پلن اشتراک | route `/plus/plans` | — | — |
| `plus.open:plus` | action | دیدن ریتمی پلاس | route `/plus/manage` | — | — |
| `plus.open:profile-credits` | action | اعتبار هوش مصنوعی | route `/plus/manage` | — | — |
| `plus.history` | action | سابقه‌ی پرداخت‌ها | route `/plus/manage` | — | — |
| `credits.pack` | single | بسته‌ی اعتبار | route `/plus/credits` | — | — |
| `credits.open:plus` | action | دیدن اشتراک | route `/plus/credits` | — | — |
| `credits.open:plus-credits` | action | خرید اعتبار | route `/profile/credits` | — | — |
| `credits.more` | action | نمایش بیشتر مصرف‌ها | route `/profile/credits` | — | — |
| `assistant.dept:*` | action | — | route `/assistant` | — | — |
| `assistant.open:assistant-profile` | action | داده‌هایی که دستیار می‌داند | route `/assistant` | — | — |
| `assistant.open:assistant` | action | بازگشت به دستیار | route `/assistant/history` | — | — |
| `assistant.open:services-doctors` | action | پیدا کردن پزشک | — | — | — |
| `assistant.open:services-doctors-all` | action | همه‌ی پزشک‌ها | — | — | — |
| `assistant.open:handoff` | action | فرستادن خلاصه برای پزشک | — | — | — |
| `assistant.history-tab:*` | action | — | route `/assistant/history` | — | — |
| `assistant.export-pdf` | action | خروجی PDF سابقه‌ی گفت‌وگوها | route `/assistant/history` | — | — |
| `assistant.share:*` | bool | — | route `/assistant/profile` | — | — |
| `assistant.profile-save` | action | ذخیره‌ی داده‌های دستیار | route `/assistant/profile` | — | — |
| `learn.open:learn-downloads` | action | دانلودها | route `/learn` | — | — |
| `learn.resume` | action | ادامه‌ی درس | route `/learn` | — | — |
| `learn.filter:*` | action | — | route `/learn` | — | — |
| `learn.course:*` | action | — | route `/learn` | — | — |
| `learn.continue` | action | ادامه‌ی دوره | — | — | — |
| `learn.course-tab:*` | action | — | — | — | — |
| `learn.lesson:*` | action | — | — | — | — |
| `learn.download:*` | action | — | — | — | — |
| `learn.play:*` | action | — | route `/learn/downloads` | — | — |
| `learn.edit-downloads` | action | ویرایش دانلودها | route `/learn/downloads` | — | — |
| `lesson.play` | action | پخش یا توقف درس | — | — | — |
| `lesson.skip-back` | action | ۱۵ ثانیه عقب | — | — | — |
| `lesson.skip-fwd` | action | ۱۵ ثانیه جلو | — | — | — |
| `lesson.speed` | action | سرعت پخش | — | — | — |
| `lesson.sleep` | action | تایمر خواب | — | — | — |
| `lesson.finish` | action | علامت‌گذاری درس به‌عنوان تمام‌شده | — | — | — |
| `lesson.seek` | integer | جابه‌جایی در درس | — | — | — |
| `lesson.notes` | text | یادداشت درس | — | — | — |
| `lesson.tab:*` | action | — | — | — | — |
| `search.query` | text | جست‌وجو در اپ | route `/search` | — | — |
| `services.search` | action | جست‌وجو در خدمات | route `/services` | — | — |
| `services.tile:*` | action | — | route `/services` | — | — |
| `articles.search` | text | جست‌وجو در مقاله‌ها | sheet `articles` | — | — |
| `articles.more` | action | مقاله‌های بیشتر | sheet `articles` | — | — |
| `companion.mute` | bool | بی‌صدا کردن پیشنهادها | route `/companion/actions` | — | — |
| `companion.code` | text | کد همدم | route `/companion/links` | — | — |
| `companion.join-submit` | action | اتصال با کد | route `/companion/links` | — | — |
| `companion.join-confirm` | action | شروع همراهی | route `/companion/links` | — | — |
| `companion.open:companion` | action | صفحه‌ی همدم | route `/companion/links` | — | — |
| `companion.open:companion-actions` | action | پیشنهادهای همدم | route `/companion` | — | — |
| `companion.open:companion-links` | action | کد همدم | route `/companion` | — | — |
| `companion.add-for` | action | افزودن برای همسر | route `/companion` | — | — |
| `companion.partner:*` | action | — | route `/companion` | — | — |
| `companion-calendar.add-medication` | action | افزودن یادآور دارو از تقویم همدم | route `/companion/calendar` | — | — |
| `companion-calendar.add-appointment` | action | افزودن نوبت پزشک از تقویم همدم | route `/companion/calendar` | — | — |
| `companion-calendar.open:companion-links` | action | باز کردن اتصال همدم از تقویم | route `/companion/calendar` | — | — |
| `companion-calendar.partner:*` | action | — | route `/companion/calendar` | — | — |
| `companion-calendar.appointment:*` | action | — | route `/companion/calendar` | — | — |
| `companion-calendar.medication:*` | action | — | route `/companion/calendar` | — | — |
| `companion.add` | action | افزودن همدم | route `/companions` | — | — |
| `companion.link:*` | action | — | route `/companions` | — | — |
| `companion.open:companions-new` | action | دعوت از همدم | — | — | — |
| `companion.goals` | multi | هدف همراهی | — | — | `cycle` `support` `low_energy` `quality_time` `info` `unknown` |
| `companion.actions` | multi | اقدام‌های ترجیحی همدم | — | — | `food` `gift` `time` `calm` `help` |
| `companion.signal:pain` | bool | نمایش «درد» به همدم | — | — | — |
| `companion.signal:mood` | bool | نمایش «حال‌وهوا» به همدم | — | — | — |
| `companion.signal:energy` | bool | نمایش «انرژی» به همدم | — | — | — |
| `companion.signal:sleep` | bool | نمایش «خواب» به همدم | — | — | — |
| `companion.period-days` | bool | نمایش «روزهای احتمالی پریود» به همدم | — | — | — |
| `companion.access:*` | single | — | — | — | — |
| `companion.child:*` | bool | — | — | — | — |
| `companion.prefs-save` | action | ذخیره‌ی ترجیحات همراهی | — | — | — |
| `companion.access-save` | action | ذخیره‌ی دسترسی‌ها | — | — | — |
| `companion.children-save` | action | ذخیره‌ی فرزندان اشتراکی | — | — | — |
| `companion.renew-invite` | action | تمدید دعوت | — | — | — |
| `invite.consent` | bool | تأیید رضایت دعوت | route `/companions/new` | — | — |
| `invite.next` | action | مرحله‌ی بعد دعوت | route `/companions/new` | — | — |
| `invite.create` | action | ساخت کد دعوت | route `/companions/new` | — | — |
| `invite.contact-toggle` | action | نام و شماره‌ی اختیاری | route `/companions/new` | — | — |
| `invite.name` | text | نام همدم | route `/companions/new` | — | — |
| `invite.phone` | text | شماره‌ی همدم | route `/companions/new` | — | — |
| `invite.preview` | action | پیش‌نمایش آنچه می‌بیند | route `/companions/new` | — | — |
| `invite.done` | action | پایان دعوت | route `/companions/new` | — | — |
| `doctors.tab:*` | single | — | route `/services/doctors` | — | — |
| `doctors.today` | bool | فقط امروز | route `/services/doctors` | — | — |
| `doctors.specialty:*` | bool | — | route `/services/doctors` | — | — |
| `doctors.city:*` | bool | — | route `/services/doctors` | — | — |
| `doctors.insurer:*` | bool | — | route `/services/doctors` | — | — |
| `doctors.prev` | action | صفحه‌ی قبل دکترها | route `/services/doctors` | — | — |
| `doctors.next` | action | صفحه‌ی بعد دکترها | route `/services/doctors` | — | — |
| `doctors.bookings` | action | نوبت‌های من | route `/services/doctors` | — | — |
| `doctor.day` | single | روز نوبت | — | — | — |
| `doctor.slot` | single | ساعت نوبت | — | — | — |
| `doctor.slots-prev` | action | هفته‌ی قبل | — | — | — |
| `doctor.slots-next` | action | هفته‌ی بعد | — | — | — |
| `doctor-profile.mode` | single | نوع ویزیت | — | — | `video` `phone` `in_person` |
| `doctor-profile.all-reviews` | action | دیدن همه‌ی نظرها | — | — | — |
| `doctor-profile.book` | action | ادامه‌ی رزرو نوبت | — | — | — |
| `doctor-book.for` | single | نوبت برای کیه | — | — | `self` `child` `other` |
| `doctor-book.child` | single | انتخاب فرزند | — | — | — |
| `doctor-book.patient` | text | نام بیمار | — | — | — |
| `doctor-book.reason` | single | دلیل ویزیت | — | — | — |
| `doctor-book.note` | text | توضیح برای دکتر | — | — | — |
| `doctor-book.share:*` | bool | — | — | — | — |
| `doctor-booked.reminders` | action | یادآورها | — | — | — |
| `doctor-booked.chat` | action | گفتگو با دکتر | — | — | — |
| `doctor-booked.home` | action | برگشت به خانه | — | — | — |
| `doctor-booked.consent:*` | bool | — | — | — | — |
| `doctor-booked.reschedule` | action | جابه‌جایی نوبت | — | — | — |
| `doctor-booked.move-confirm` | action | تأیید زمان جدید | — | — | — |
| `doctor-chat.add-reminder:*` | action | — | — | — | — |
| `doctor-bookings.tab:*` | single | — | route `/services/bookings` | — | — |
| `doctor-bookings.find` | action | پیدا کردن دکتر | route `/services/bookings` | — | — |
| `home-lab.service:*` | action | — | route `/services/home-lab` | — | — |
| `home-lab.request:*` | action | — | route `/services/home-lab` | — | — |
| `home-lab.card:*` | action | — | — | — | — |
| `home-lab-form.field:*` | text | — | — | — | — |
| `home-lab-form.day` | single | روز نمونه‌گیری | — | — | — |
| `home-lab-form.window` | single | بازه‌ی زمانی | — | — | `morning` `noon` `evening` |
| `home-lab-form.consent-share` | bool | رضایت برای ارسال اطلاعات | — | — | — |
| `home-lab-form.consent-import` | bool | رضایت برای ثبت نتیجه در پرونده | — | — | — |
| `home-lab-request.consent` | bool | رضایت برای ثبت نتیجه | — | — | — |
| `home-lab-request.import` | action | ثبت نتیجه در پرونده | — | — | — |
| `home-lab-request.new` | action | درخواست تازه | — | — | — |
| `city-home.city` | action | انتخاب شهر | route `/city-services` | — | — |
| `city-home.city-pick` | single | شهر | route `/city-services` | yes | — |
| `city-home.search` | text | جست‌وجوی مکان‌ها | route `/city-services` | — | — |
| `city-home.category:*` | action | — | route `/city-services` | — | — |
| `city-home.listing:*` | action | — | route `/city-services` | — | — |
| `city-home.rail:*` | action | — | route `/city-services` | — | — |
| `city-home.bookings` | action | نوبت‌های من | route `/city-services` | — | — |
| `city-home.map` | action | نقشه‌ی مکان‌ها | route `/city-services` | — | — |
| `city-home.map-all` | action | نقشه | route `/city-services` | — | — |
| `city-home.next-booking` | action | نوبت بعدی | route `/city-services` | — | — |
| `city-home.suited-all` | action | همه‌ی مناسب‌ها | route `/city-services` | — | — |
| `city-home.list` | action | فهرست مکان‌ها | route `/city-services` | — | — |
| `city-home.join` | action | ثبت مکان من | route `/city-services` | — | — |
| `city-home.age-toggle` | action | فعال یا غیرفعال کردن سن کودک | route `/city-services` | — | — |
| `city-list.search` | text | جست‌وجو در فهرست | route `/city-services` | yes | — |
| `city-list.filters` | action | فیلترها | route `/city-services` | yes | — |
| `city-list.open-now` | bool | فقط بازها | route `/city-services` | yes | — |
| `city-list.city` | single | شهر فهرست | route `/city-services` | yes | — |
| `city-list.category` | single | دسته‌ی مکان | route `/city-services` | yes | — |
| `city-list.prev` | action | صفحه‌ی قبل مکان‌ها | route `/city-services` | yes | — |
| `city-list.next` | action | صفحه‌ی بعد مکان‌ها | route `/city-services` | yes | — |
| `city-list.bookings` | action | نوبت‌های من | route `/city-services` | yes | — |
| `city-list.map` | action | نقشه | route `/city-services` | yes | — |
| `city-list.listing:*` | action | — | route `/city-services` | — | — |
| `city-filter.age` | single | سن کودک | route `/city-services` | yes | — |
| `city-filter.open-now` | bool | فقط بازها | route `/city-services` | yes | — |
| `city-filter.amenities` | multi | امکانات | route `/city-services` | yes | — |
| `city-filter.sort` | single | مرتب‌سازی | route `/city-services` | yes | `rank` `distance` `rating` |
| `city-filter.clear` | action | پاک کردن فیلترها | route `/city-services` | yes | — |
| `city-filter.apply` | action | اعمال فیلترها | route `/city-services` | yes | — |
| `city-detail.service:*` | bool | — | — | — | — |
| `city-detail.slot:*` | bool | — | — | — | — |
| `city-detail.continue` | action | ادامه‌ی رزرو | — | — | — |
| `city-detail.bookings` | action | نوبت‌های من | — | — | — |
| `city-detail.join` | action | ثبت مکان من | — | — | — |
| `city-detail.report` | action | گزارش اطلاعات نادرست | — | — | — |
| `city-detail.report-reason` | single | دلیل گزارش | — | yes | `wrong_info` `closed` `wrong_location` `other` |
| `city-detail.report-note` | text | توضیح گزارش | — | yes | — |
| `city-detail.report-send` | action | ارسال گزارش | — | yes | — |
| `city-book.service` | single | خدمت | — | — | — |
| `city-book.child` | single | کودک | — | — | — |
| `city-book.day` | single | روز | — | — | — |
| `city-book.slot` | single | ساعت | — | — | — |
| `city-book.name` | text | نام تماس | — | — | — |
| `city-book.note` | text | توضیح برای مکان | — | — | — |
| `city-book.submit` | action | ارسال درخواست نوبت | — | — | — |
| `city-request.field:*` | text | — | — | — | — |
| `city-request.day` | single | روز دلخواه | — | — | — |
| `city-request.age` | single | سن کودک | — | — | `none` `0-1` `1-3` `3-6` `6-12` `12+` |
| `city-request.note` | text | توضیح درخواست | — | — | — |
| `city-request.submit` | action | ارسال درخواست | — | — | — |
| `city-booked.review` | action | ثبت نظر | — | — | — |
| `city-review.rating` | single | امتیاز | — | — | `1` `2` `3` `4` `5` |
| `city-review.body` | text | متن نظر | — | — | — |
| `city-review.send` | action | ارسال نظر | — | — | — |
| `city-bookings.tab:*` | single | — | route `/profile/bookings` | — | — |
| `city-bookings.open:*` | action | — | route `/profile/bookings` | — | — |
| `city-bookings.reschedule:*` | action | — | route `/profile/bookings` | — | — |
| `city-bookings.review:*` | action | — | route `/profile/bookings` | — | — |
| `city-bookings.again:*` | action | — | route `/profile/bookings` | — | — |
| `city-bookings.browse` | action | دیدن مکان‌ها | route `/profile/bookings` | — | — |
| `city-bookings.doctors` | action | نوبت‌های دکتر | route `/profile/bookings` | — | — |
| `city-join.business-name` | text | نام مجموعه | route `/city-services/join` | — | — |
| `city-join.kind` | multi | نوع مجموعه | route `/city-services/join` | — | `class` `pool` `playhouse` `music_art` `postpartum_fitness` `baby_massage` `parenting_workshop` `other` |
| `city-join.city` | single | شهر مجموعه | route `/city-services/join` | — | — |
| `city-join.district` | text | محله | route `/city-services/join` | — | — |
| `city-join.contact-name` | text | نام رابط | route `/city-services/join` | — | — |
| `city-join.contact-phone` | text | تلفن رابط | route `/city-services/join` | — | — |
| `city-join.owner` | bool | تأیید مالکیت | route `/city-services/join` | — | — |
| `city-join.next:*` | action | — | route `/city-services/join` | — | — |
| `city-join.address` | text | آدرس | route `/city-services/join` | yes | — |
| `city-join.about` | text | درباره‌ی مجموعه | route `/city-services/join` | yes | — |
| `city-join.ages` | multi | رده‌های سنی | route `/city-services/join` | yes | `pregnancy` `m0_6` `m6_12` `y1_2` `y2_4` `y4_6` |
| `city-join.amenities` | multi | امکانات | route `/city-services/join` | yes | `nursing_room` `changing_table` `female_instructor` `stroller_parking` `parking` `elevator` `wheelchair_access` |
| `city-join.hours-open-day:*` | bool | — | — | — | — |
| `city-join.hours-open:*` | time | — | — | — | — |
| `city-join.hours-close:*` | time | — | — | — | — |
| `city-join.service-add` | action | افزودن خدمت | route `/city-services/join` | yes | — |
| `city-join.service-kind` | single | نوع خدمت | route `/city-services/join` | yes | `session` `package` |
| `city-join.service-name` | text | نام خدمت | route `/city-services/join` | yes | — |
| `city-join.service-desc` | text | توضیح خدمت | route `/city-services/join` | yes | — |
| `city-join.service-format` | single | شکل برگزاری | route `/city-services/join` | yes | `group` `private` |
| `city-join.service-field:*` | text | — | route `/city-services/join` | — | — |
| `city-join.service-save` | action | ذخیره‌ی خدمت | route `/city-services/join` | yes | — |
| `city-join.booking-mode` | single | روش رزرو | route `/city-services/join` | yes | `online` `request` `phone` |
| `city-join.terms` | bool | پذیرش قوانین | route `/city-services/join` | yes | — |
| `city-join.submit` | action | ارسال درخواست همکاری | route `/city-services/join` | yes | — |
| `city-join.new` | action | درخواست تازه | route `/city-services/join` | — | — |
| `city-join.home` | action | برگشت به خانه | route `/city-services/join` | — | — |
| `insurance.action:*` | action | — | route `/insurance` | — | — |
| `insurance.claims-all` | action | همه‌ی خسارت‌ها | route `/insurance` | — | — |
| `insurance.current` | action | برگشت به بیمه‌نامه‌ی فعلی | route `/insurance` | — | — |
| `insurance.attention` | action | خسارت‌های نیازمند اقدام | route `/insurance` | — | — |
| `insurance.coverage` | action | جزئیات پوشش | route `/insurance` | — | — |
| `insurance.renew` | action | تمدید بیمه‌نامه | route `/insurance` | — | — |
| `insurance.edit` | action | ویرایش بیمه‌نامه | route `/insurance` | — | — |
| `insurance.info` | action | راهنمای بیمه | route `/insurance` | — | — |
| `insurance.claim:*` | action | — | route `/insurance` | — | — |
| `insurance.policy:*` | bool | — | route `/insurance` | — | — |
| `insurance-info.faq:*` | section | — | — | — | — |
| `ins-setup.insurer` | single | بیمه‌گر | route `/insurance` | yes | — |
| `ins-setup.insurer-name` | text | نام بیمه‌گر | route `/insurance` | yes | — |
| `ins-setup.plan` | text | نام طرح | route `/insurance` | yes | — |
| `ins-setup.number` | text | شماره‌ی بیمه‌نامه | route `/insurance` | yes | — |
| `ins-setup.date:validFrom` | action | تاریخ شروع | route `/insurance` | yes | — |
| `ins-setup.date:validTo` | action | تاریخ پایان | route `/insurance` | yes | — |
| `ins-setup.calendar` | date | تاریخ بیمه‌نامه | route `/insurance` | yes | — |
| `ins-setup.date-done` | action | تأیید تاریخ | route `/insurance` | yes | — |
| `ins-setup.deadline` | integer | مهلت اعلام خسارت | route `/insurance` | yes | — |
| `ins-setup.member-kind:*` | single | — | route `/insurance` | — | — |
| `ins-setup.member-name:*` | text | — | route `/insurance` | — | — |
| `ins-setup.member-add` | action | افزودن عضو | route `/insurance` | yes | — |
| `ins-setup.cap:*` | number | — | route `/insurance` | — | — |
| `ins-setup.deductible:*` | number | — | route `/insurance` | — | — |
| `ins-setup.waiting:*` | number | — | route `/insurance` | — | — |
| `ins-setup.save` | action | ذخیره‌ی بیمه‌نامه | route `/insurance` | yes | — |
| `ins-coverage.edit` | action | ویرایش پوشش | route `/insurance/coverage` | — | — |
| `ins-centers.search` | text | جست‌وجوی مراکز | route `/insurance/centers` | — | — |
| `ins-centers.type` | single | نوع مرکز | route `/insurance/centers` | — | `gyn` `lab` `imaging` `hospital` `pharmacy` `pediatric` |
| `ins-claims.search` | text | جست‌وجوی خسارت‌ها | route `/insurance/claims` | — | — |
| `ins-claims.filter` | single | فیلتر خسارت‌ها | route `/insurance/claims` | — | `all` `open` `paid` `rejected` |
| `ins-claims.add` | action | ثبت خسارت تازه | route `/insurance/claims` | — | — |
| `ins-claims.open:*` | action | — | route `/insurance/claims` | — | — |
| `ins-claim-new.who` | single | خسارت برای کیه | route `/insurance/claims/new` | — | — |
| `ins-claim-new.cost-type` | single | نوع هزینه | route `/insurance/claims/new` | — | `paraclinic` `visit` `drug` `inpatient` `delivery` `dental` `optical` `infertility` `other` |
| `ins-claim-new.centre` | text | نام مرکز درمانی | route `/insurance/claims/new` | — | — |
| `ins-claim-new.date` | action | تاریخ خدمت | route `/insurance/claims/new` | — | — |
| `ins-claim-new.calendar` | date | تاریخ خدمت | route `/insurance/claims/new` | yes | — |
| `ins-claim-new.date-done` | action | تأیید تاریخ | route `/insurance/claims/new` | yes | — |
| `ins-claim-new.amount` | number | مبلغ فاکتور | route `/insurance/claims/new` | — | — |
| `ins-claim-new.doc-add:*` | action | — | route `/insurance/claims/new` | — | — |
| `ins-claim-new.submit` | action | ثبت خسارت | route `/insurance/claims/new` | — | — |
| `ins-docs.pick:*` | action | — | — | — | — |
| `ins-claim.attach` | action | پیوست مدرک از پرونده | — | — | — |
| `ins-claim.advance` | action | ثبت مرحله‌ی تازه | — | — | — |
| `ins-claim.appeal` | action | اعتراض به نتیجه | — | — | — |
| `ins-claim.step-status` | single | وضعیت مرحله | — | yes | `reviewing` `docs_requested` `paid` `rejected` `appealed` |
| `ins-claim.step-amount` | number | مبلغ پرداخت‌شده | — | yes | — |
| `ins-claim.step-note` | text | یادداشت مرحله | — | yes | — |
| `ins-claim.step-save` | action | ذخیره‌ی مرحله | — | yes | — |
| `ins-quest.purpose` | single | هدف پرسش‌نامه | route `/insurance/questionnaire` | — | `issue` `renewal` |
| `ins-quest.begin` | action | شروع پرسش‌نامه | route `/insurance/questionnaire` | — | — |
| `ins-quest.answer:*` | single | — | route `/insurance/questionnaire` | — | — |
| `ins-quest.explanation` | text | توضیح تکمیلی | route `/insurance/questionnaire` | — | — |
| `ins-quest.next` | action | مرحله‌ی بعد | route `/insurance/questionnaire` | — | — |
| `ins-quest.prev` | action | مرحله‌ی قبل | route `/insurance/questionnaire` | — | — |
| `ins-quest.confirm` | action | تأیید پرسش‌نامه | route `/insurance/questionnaire` | — | — |
| `ins-quest.edit-section:*` | action | — | route `/insurance/questionnaire` | — | — |
| `ins-quest.edit` | action | ویرایش پرسش‌نامه | route `/insurance/questionnaire` | — | — |
| `ins-quest.status` | action | وضعیت درخواست | route `/insurance/questionnaire` | — | — |
| `ins-status.begin` | action | شروع پرسش‌نامه | route `/insurance/status` | — | — |
| `ins-status.claims-all` | action | همه‌ی خسارت‌ها | route `/insurance/status` | — | — |
| `ins-status.claim:*` | action | — | route `/insurance/status` | — | — |
| `ins-status.continue:*` | action | — | route `/insurance/status` | — | — |
| `ins-status.record:*` | action | — | route `/insurance/status` | — | — |
| `ins-status.step` | single | مرحله‌ی درخواست | route `/insurance/status` | — | `submitted` `reviewing` `doctor_review` `approved` `conditional` `rejected` |
| `ins-status.date` | date | تاریخ مرحله | route `/insurance/status` | — | — |
| `ins-status.record-save` | action | ذخیره‌ی مرحله | route `/insurance/status` | — | — |
| `shop.cart` | action | سبد خرید | — | — | — |
| `shop.product:*` | action | — | — | — | — |
| `shop-index.category:*` | action | — | route `/shop` | — | — |
| `shop-home.root:*` | single | — | — | — | — |
| `shop-home.category:*` | action | — | — | — | — |
| `shop-home.all:*` | action | — | — | — | — |
| `shop-home.checklist` | action | چک‌لیست خرید | — | — | — |
| `shop-home.supply` | bool | یادآور لوازم شخصی | — | — | — |
| `shop-category.product:*` | section | — | — | — | — |
| `shop-list.category:*` | action | — | — | — | — |
| `shop-list.search` | text | جست‌وجوی کالا | — | — | — |
| `shop-list.filters` | action | فیلترهای کالا | — | — | — |
| `shop-list.sort` | single | مرتب‌سازی کالا | — | — | `popular` `rating` `newest` `price_asc` `price_desc` |
| `shop-list.prev` | action | صفحه‌ی قبل کالاها | — | — | — |
| `shop-list.next` | action | صفحه‌ی بعد کالاها | — | — | — |
| `shop-filter.facet:*` | multi | — | — | — | — |
| `shop-filter.price-min` | text | حداقل قیمت | — | yes | — |
| `shop-filter.price-max` | text | حداکثر قیمت | — | yes | — |
| `shop-filter.reset` | action | پاک کردن فیلترها | — | yes | — |
| `shop-filter.apply` | action | اعمال فیلترها | — | yes | — |
| `shop-product.option:*` | single | — | — | — | — |
| `shop-product.add-to-cart` | action | افزودن به سبد | — | — | — |
| `shop-product.view-cart` | action | دیدن سبد | — | — | — |
| `shop-product.all-reviews` | action | همه‌ی نظرها | — | — | — |
| `shop-cart.inc:*` | action | — | route `/shop/cart` | — | — |
| `shop-cart.dec:*` | action | — | route `/shop/cart` | — | — |
| `shop-cart.product:*` | action | — | route `/shop/cart` | — | — |
| `shop-cart.discount-code` | text | کد تخفیف | route `/shop/cart` | — | — |
| `shop-cart.discount-apply` | action | اعمال کد تخفیف | route `/shop/cart` | — | — |
| `shop-cart.discount-remove` | action | برداشتن کد تخفیف | route `/shop/cart` | — | — |
| `shop-cart.continue` | action | ادامه‌ی خرید | route `/shop/cart` | — | — |
| `shop-checklist.share` | action | اشتراک چک‌لیست | route `/shop/checklist` | — | — |
| `shop-checklist.add` | action | افزودن آیتم به چک‌لیست | route `/shop/checklist` | — | — |
| `shop-checklist.filter` | single | فیلتر چک‌لیست | route `/shop/checklist` | — | — |
| `shop-checklist.have:*` | bool | — | route `/shop/checklist` | — | — |
| `shop-checklist.see:*` | action | — | route `/shop/checklist` | — | — |
| `shop-checklist.edit:*` | action | — | route `/shop/checklist` | — | — |
| `shop-checklist.title` | text | نام آیتم | route `/shop/checklist` | yes | — |
| `shop-checklist.group` | single | گروه آیتم | route `/shop/checklist` | yes | `clothing` `sleep` `feeding` `bath` `outing` |
| `shop-checklist.state` | single | وضعیت آیتم | route `/shop/checklist` | yes | `todo` `have` `in_cart` `on_the_way` `not_needed` `later` |
| `shop-checklist.qty` | integer | تعداد | route `/shop/checklist` | yes | — |
| `shop-checklist.save` | action | ذخیره‌ی آیتم | route `/shop/checklist` | yes | — |
| `shop-order.return:*` | action | — | — | — | — |
| `shop-order.review:*` | action | — | — | — | — |
| `shop-order.my-orders` | action | سفارش‌های من | — | — | — |
| `shop-order.shop` | action | برگشت به فروشگاه | — | — | — |
| `shop-return.qty` | integer | تعداد مرجوعی | — | — | — |
| `shop-return.reason` | single | دلیل مرجوعی | — | — | `wrong_size` `damaged` `not_as_described` `changed_mind` `other` |
| `shop-return.note` | text | توضیح مرجوعی | — | — | — |
| `shop-return.submit` | action | ثبت درخواست مرجوعی | — | — | — |
| `shop-review.rating` | single | امتیاز کالا | — | — | `1` `2` `3` `4` `5` |
| `shop-review.body` | text | متن نظر | — | — | — |
| `shop-review.submit` | action | ثبت نظر | — | — | — |
| `my-orders.browse` | action | دیدن فروشگاه | route `/profile/orders` | — | — |
| `my-orders.open:*` | action | — | route `/profile/orders` | — | — |
