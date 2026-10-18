# Ritmi Persian glossary (M8)

One canonical term per concept. Counts = occurrences in `frontend/messages/fa/*.json` on 2026-10-18 (before
M8). The checker (`frontend/scripts/copy-fa/check.mjs`) parses the **Rejected** column and warns on each hit;
put several rejected variants in one cell separated by «، ». Fixed medical expressions in the **Allowed in**
column are not flagged.

Status: **approved by the user 2026-10-18** (چرخه over سیکل; پریود everywhere, قاعدگی only in fixed medical phrases).

## Cycle and body

| Concept | Canonical | Rejected | Allowed in / notes | Counts |
|---|---|---|---|---|
| menstruation (everyday) | پریود | قاعدگی، دوره (for period)، عادت ماهانه | «قاعدگی» only in fixed medical phrases: «آخرین قاعدگی (LMP)»، «پیش از قاعدگی (PMS/PMDD)»، «اختلال قاعدگی». Phase label = «پریود». | پریود 192 · قاعدگی 13 · دوره 2+ |
| menstrual cycle | چرخه | سیکل | «چرخهٔ قاعدگی» in medical text | سیکل 258 · چرخه 45 |
| fertile window | پنجرهٔ باروری | پنجره باروری، روزهای باروری (as the name) | | 29 (all without ezafe) |
| ovulation | تخمک‌گذاری | تخمک گذاری | | 54 |
| luteal phase | فاز لوتئال | لوتیال | | لوتئال 19 · لوتیال 2 |
| follicular phase | فاز فولیکولار | فولیکولی | | 12 |
| spotting | لکه‌بینی | اسپاتینگ | | 29 |
| discharge | ترشحات | | | 11 |
| symptoms | علائم | علایم | | 128 |
| mood | حال (روحی) / خلق‌وخو | حال و هوا، مود | «نوسان خلق‌وخو» | |
| basal body temperature | دمای پایهٔ بدن (BBT) | دمای پایه بدن | short label «دمای پایه» OK in chips | 36 |
| LH test | تست LH | آزمایش LH | lead with «تست», never start a value with «LH» | 22 |

## Pregnancy, postpartum, family

| Concept | Canonical | Rejected | Notes | Counts |
|---|---|---|---|---|
| pregnancy | بارداری | حاملگی | | 194 · 1 |
| week N | هفتهٔ {n} | هفته {n}، هفته‌ی {n} | | |
| ultrasound | سونوگرافی | سونو | «سونو» allowed only in chips ≤ 6 chars if the layout needs it — list it | 22 · 35 |
| birth / delivery | زایمان | | | 81 |
| postpartum | پس از زایمان | بعد از زایمان (as the mode/section name) | «بعد از زایمان» fine in running prose | 7 · 22 |
| breastfeeding | شیردهی | | | 28 |
| baby (0–1 y) / child | نوزاد / کودک | بچه | | |
| pregnancy loss | از دست دادن بارداری | سقط (in UI headings) | «سقط» only where the medical term is required (e.g. «سقط فراموش‌شده» in clinical lists) | |
| IVF | IVF / آی‌وی‌اف | | never start a value with «IVF»: «درمان IVF …» | 6 |
| partner / companion | همراه | شریک | «همسر» only where the text is explicitly about a spouse | همراه 40 · شریک 16 · همسر 11 |

## Health record and care

| Concept | Canonical | Rejected | Notes |
|---|---|---|---|
| doctor | پزشک | دکتر | «دکتر» only as a title before a name («دکتر رضایی») |
| gynaecologist | متخصص زنان | | |
| lab test | آزمایش | تست (except home/rapid tests: تست بارداری، تست LH) | |
| checkup | چکاپ | | «غربالگری» for screening programmes |
| medication | دارو | | «قرص» only for pills specifically (قرص ضدبارداری) |
| appointment | نوبت | ویزیت (as the noun for the booking) | «ویزیت» OK for the visit itself |
| reminder | یادآور | یادآوری (as the item noun) | «یادآوری» = the act («یادآوری روزانه») |
| health record | پروندهٔ سلامت | پرونده سلامت، سوابق (as the section name) | |
| blood pressure / glucose | فشار خون / قند خون | | |
| pelvic floor | کف لگن | | |
| menopause / perimenopause | یائسگی / پیش‌یائسگی | | |

## App UI vocabulary

| Concept | Canonical | Rejected | Notes |
|---|---|---|---|
| brand | ریتمی | ریتمی‌اپ، Ritme in Persian text | |
| log (verb / noun) | ثبت / ثبت کردن | لاگ | |
| save | ذخیره | | button «ذخیره» |
| cancel (dismiss a form/dialog) | انصراف | لغو | «لغو» = cancel a booking/subscription/order |
| confirm | تأیید | تایید | |
| continue / next | ادامه (flows) / بعدی (step N → N+1 in a wizard) | | |
| back | بازگشت | برگشت، برگرد | aria-label for back button = «بازگشت» |
| close | بستن | | |
| edit / delete | ویرایش / حذف | پاک کردن (as button) | «پاک می‌شود» fine in explanations |
| retry | دوباره امتحان کن | دوباره تلاش کن، مجدداً | |
| profile | پروفایل | نمایه | |
| account | حساب | اکانت، حساب کاربری (in buttons) | |
| notification | اعلان | نوتیفیکیشن | |
| subscription / Plus | اشتراک / ریتمی پلاس | | |
| report / analysis | گزارش / تحلیل | آنالیز | |
| chart | نمودار | | |
| price | … تومان | ریال | always «تومان» |
| update (app) | به‌روزرسانی | آپدیت | |
| today | امروز | | |
