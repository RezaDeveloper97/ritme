# M8 Persian copy — visual QA (T-M8-15)

Date: 2026-10-18. Local stack: backend-go on :8020 against `ritme_dev` (contract fixtures `dump.sql` + goose
00002…00094), Next.js dev on :3000, Playwright at 390×844 (mobile), light and dark theme.
Screenshots: [`qa/`](qa/) — `<theme>-<persona>-<route>.png` (50 files).

Personas: `cycle` = 09900000004 (regular cycle, 6 days overdue on the QA date), `ttc` = 09900000012,
`preg` = 09900000014 (pregnant). `anon` = signed out.

## Automated check

For every screen the script walked all buttons, links, tabs, chips, `[class*=btn]`, headings and labels with
Persian text, and flagged (a) an element whose content is wider than its box (`scrollWidth > clientWidth` with
clipping overflow) and (b) an element outside the viewport; it also flagged any horizontal page scroll.

| Result | Screens |
|---|---|
| Horizontal page scroll | **none** (0 of 50) |
| Clipped button/chip/tab/heading text | **none** — the only `over` hits are scroll containers (analysis filter row, a hidden `0×1` heading in `record`) |
| Outside the viewport | only by design: welcome carousel slides, services/search/analysis chip rows that scroll sideways |

Screens covered (light + dark): welcome, signup, home (cycle + pregnancy), calendar, cycle, profile («من»),
services, analysis, log sheet (`?sheet=log`), reminders, checkups, labs, record, insurance, shop, plus, learn,
search, city-services, companions, fertility, ivf, pregnancy, pregnancy/weeks.

## Manual review (screenshots read)

- **home** (cycle): «پریودم شروع شد»، «روز ۳۵ چرخه»، late-period card «پریودت ۶ روز از زمان پیش‌بینی‌شده
  گذشته است… تأخیر چندروزه معمولاً طبیعی است.» — register right, hedge kept, no wrap problems.
- **log sheet**: «ثبت امروز»، «ثبت دستی / ثبت با صدا»، chips «پریود، درد، حال، علائم، خواب، ترشحات، وزن،
  یادداشت»، «نوبت یا ویزیت»، «کار جدید» — chips fit.
- **analysis**: «مهم‌ترین یافتهٔ ۶ ماه»، «چرخه‌هایت منظم‌اند (نوسان ۰ روز).»، «طول چرخه و پریود»، «پریود و
  تاریخچهٔ چرخه‌ها». (Month abbreviations under the bar chart are cut to two letters — pre-existing chart
  layout, not copy.)
- **pregnancy** (dark): «هفتهٔ ۱۲ از ۴۰»، «روز تا زایمان»، «سه‌ماههٔ اول»، «تاریخ تقریبی زایمان… این یک
  تخمین است، نه تشخیص پزشکی.»
- **record**: «پروندهٔ سلامت من»، «آخرین به‌روزرسانی: ۱ مهر»، «همهٔ اسناد»، «سابقهٔ خانوادگی».
- **ivf** (module off for this persona): «پیگیری درمان ناباروری خاموش است… را در مرحلهٔ زندگی روشن کن.»
- **welcome**: «چرخه‌ات را بشناس» in the display font — ZWNJ and ezafe «هٔ» render correctly in Lalezar and
  Vazirmatn (no tofu, no detached letters).
- **profile** (dark): «همدم‌ها و خانواده»، «حالت اپ · تغییر مرحلهٔ زندگی»؛ PWA install card «ریتمی را به
  صفحهٔ اصلی گوشی‌ات اضافه کن تا سریع‌تر به آن برسی.»

## Not covered here

- `ivf.actions.scan` («ثبت نتیجهٔ سونوگرافی») — needs a persona with the IVF module on and an open treatment
  cycle; the fixtures have none.
- `city-join` provider flow buttons («دوباره امتحان کنید»، «نگه دارید») — needs a business sign-up draft.
- English screens (only `profile-info` en changed).

Verdict: no copy-induced layout regression found on the covered screens.
