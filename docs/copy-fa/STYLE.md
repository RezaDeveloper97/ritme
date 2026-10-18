# Ritmi Persian copy — style guide (M8)

The contract for every rewrite of `frontend/messages/fa/*.json` (tasks T-M8-03 … T-M8-15).
Source of the writing rules: the `persian-writing` skill (`references/writing-style.md`,
`references/orthography.md`). Terms: [GLOSSARY.md](GLOSSARY.md). Batches: [INVENTORY.md](INVENTORY.md).

Status: **approved by the user 2026-10-18**, amended after the T-M8-03 pilot (same day: بارگذاری, error titles, parentheses for asides, همدم/شریک) (register «تو» written-but-warm, scope = frontend, glossary: چرخه + پریود).

---

## 1. Register — one voice for the whole app

**Written-but-warm, addressing the reader as «تو».** (User decision, 2026-10-18.)

Ritmi talks like a knowledgeable friend who writes carefully: full written verb forms, second-person
singular, short sentences, no slang, no office Persian.

| Rule | ✅ | ❌ |
|---|---|---|
| Full written verbs | می‌شود، می‌کند، می‌رود، آمد، می‌توانی | میشه، می‌کنه، می‌ره، اومد، می‌تونی |
| Object marker «را» | چرخه‌ات را ببین | چرخه‌ات رو ببین |
| «است» in full (attached «ـست» only after ا/و: «آماده‌ست» ❌، «جاست» ✅) | انرژی معمولاً پایین‌تر است، همین روز است | پایین‌تره، همین روزه |
| Written pronouns/words | این، آن، یک، کمی، برایت، به تو، در، این‌جا | اون، یه، یه‌کم، برات، بهت، توی |
| Second person singular | بدنت، می‌توانی، ثبت کن | بدنتان، می‌توانید، ثبت کنید |
| Conditional | اگر | اگه |
| No office Persian | است، شد، انجام شد | می‌باشد، گردید، به عمل آمد، لازم به ذکر است، در راستای |

«تو» is carried by the verb and the possessive suffix; don't write the pronoun «تو» unless it's needed
for emphasis («این تصمیم با توست»).

### 1.1 Exceptions to «تو»

| Audience | Address | Namespaces |
|---|---|---|
| The user (incl. teen, partner/companion using the app) | «تو» | everything by default |
| A business / clinic / provider signing up or managing a listing | «شما» (formal-but-human) | `city-join`, `web-business`, provider-facing keys in `web-directory` |
| Legal text the user agrees to (terms, consent clauses) | «تو» but formal vocabulary, no warmth idioms | `lab-consent`, consent keys in `auth`/`onboarding` |

### 1.2 «ما» (point of view)

«ما» only for what Ritmi itself does or is responsible for: «برایت پیامک می‌کنیم»، «داده‌هایت را با کسی به
اشتراک نمی‌گذاریم». Facts about the body are impersonal («انرژی معمولاً پایین‌تر است»), not «ما فکر
می‌کنیم». Don't mix «ریتمی … می‌کند» and «ما … می‌کنیم» in one string.

---

## 2. Rules per string kind

| Kind (typical key names) | Rule | Example |
|---|---|---|
| Button / CTA (`cta`, `button`, `submit`, `save`, `action`) | Imperative, 1–3 words, no period, never longer than now | «ذخیره»، «ادامه»، «حالت بارداری را روشن کن» |
| Tab / chip / nav (`tab`, `chip`, `nav.*`, taxonomy) | Noun, 1–2 words, glossary term | «تقویم»، «لکه‌بینی» |
| Screen / sheet title (`title`) | Noun phrase or short sentence, no period | «شماره‌ی موبایلت» → «شمارهٔ موبایلت» |
| Subtitle / body | Full sentences ending with «.»; one idea per sentence | |
| Field label / placeholder | Noun; placeholder may be an example, no period | |
| Empty state (`empty`) | What's missing + what to do, ≤ 2 sentences | «هنوز چیزی ثبت نکرده‌ای. از دکمهٔ + شروع کن.» |
| Loading (`loading`, `updating`) | Present progressive + «…» | «در حال به‌روزرسانی چرخه‌ات…» |
| Error / toast (`error`, `failed`) | §2.1 templates | |
| Success toast (`saved`, `success`) | Past tense, short, no «!» | «ذخیره شد.» |
| Health explanation (phase, insight, tip, alert) | §4 medical rules | |
| Confirmation dialog (`confirm`, `delete*`) | Say exactly what happens and whether it's reversible | «فایل‌ها و اطلاعات این سند برای همیشه پاک می‌شوند.» |

Exclamation marks: at most one, only in welcome/celebration copy; never in errors, health, loss, IVF.

### 2.1 Error templates (use these everywhere; consistency beats variety here)

| Situation | Text |
|---|---|
| Load failed | «‹X› بارگذاری نشد. کمی بعد دوباره امتحان کن.» As a title/heading: «‹X› بارگذاری نشد» (no period) |
| Save failed | «ذخیره نشد. کمی بعد دوباره امتحان کن.» |
| Delete failed | «حذف نشد. دوباره امتحان کن.» |
| Generic | «مشکلی پیش آمد. دوباره امتحان کن.» |
| Network | «اتصال برقرار نشد. اینترنتت را بررسی کن و دوباره امتحان کن.» |
| Rate limit (429) | «چند لحظه صبر کن و دوباره امتحان کن.» (keep any extra reason the current string gives) |
| Not allowed for future dates | «برای روزهای آینده نمی‌توانی ثبت کنی.» |
| A named action failed | «‹X› نشد. دوباره امتحان کن.» («حساب حذف نشد.»، «اطلاعاتت دریافت نشد.»); consent/agreement = «ثبت نشد» |

Titles and buttons never end with «.»; body sentences and toasts always do.

---

## 3. Orthography (mechanical — enforced by `frontend/scripts/copy-fa/check.mjs`)

1. **ZWNJ** (U+200C): می‌/نمی‌ + verb, ‌ها/‌های, ‌تر/‌ترین, ‌ام/‌ات/‌اش after ه or ی, ‌ای after ه,
   compounds (به‌روزرسانی، هفته‌به‌هفته، پیش‌بینی، تخمک‌گذاری). Lexicalised words stay solid: بیشتر، کمتر،
   بهتر، همکار.
2. **Ezafe after silent ه: «هٔ»** (همهٔ اطلاعاتت، شمارهٔ موبایل) — the current majority form (108 vs 21) and the
   default of `persian_cleanup.py --edit`. Convert the 21 «ه‌ی» to «هٔ». Never «ـه» for ezafe (هکسره).
3. **Persian characters**: ی U+06CC, ک U+06A9, never ي ك ة.
4. **Digits**: Persian digits in prose («۴۸ ساعت»، «۱. توی نوار …»). ICU placeholders (`{n}`, `{n, number}`)
   are formatted by the locale and stay untouched. Latin digits only inside Latin codes/units that stay LTR
   (HbA1c, B12, mg/dL, IVF, PMDD, «Android 12»). Percent «۲۰٪».
5. **Punctuation**: ، ؛ ؟ «…» quotes; no space before, one after; ellipsis is the single character «…»; no
   em/en dash (—, –) in prose: use «،» «؛» «:» or parentheses or two sentences. A value that is exactly «—»
   (an empty-value placeholder) is allowed. A dash-delimited aside becomes parentheses: «همهٔ اطلاعاتت (پروفایل، …) پاک می‌شود»; a dash between label and explanation becomes «:» or «؛», and so is an en dash between two numbers or ICU arguments (`{min}–{max}`, `۲–۳`).
6. **Never start a value with a Latin word** (it flips direction). «آزمایش TSH …»، not «TSH …».
   Exception: a value that *is* only a code/unit (`"mg/dL"`).
7. **Tanvin** only on Arabic loans (لطفاً، حتماً، معمولاً، تقریباً); never گاهاً. «لطفا» → «لطفاً».
8. One space between words, none inside «» or ().

---

## 4. Medical and sensitive content — safety rules (non-negotiable)

Rewriting is about wording only. A rewrite that changes medical meaning is a bug, not an improvement.

1. **Never change** a number, unit, range, threshold, dose, frequency, time window («۴۸ ساعت»، «هفتهٔ ۲۰
   تا ۲۴»), or the list of symptoms in a warning.
2. **Keep every hedge** at its strength: معمولاً، ممکن است، احتمالاً، تقریباً، در بیشتر موارد. Don't upgrade
   «ممکن است» to «می‌شود», don't downgrade «فوراً» to «بهتر است».
3. **Keep urgency** of alert copy («همین حالا با پزشک تماس بگیر»، «به اورژانس برو»).
4. **Never add a claim**, a statistic, a benefit or a reassurance that the current text (or `en`) doesn't make.
5. Predictions stay predictions: «پیش‌بینی»، «تخمینی»، «احتمالاً» — never present a predicted date as fact.
6. When fa and en disagree in meaning, **do not resolve it**: leave the fa string, list it in the batch report.
7. Uncertain about nuance → leave unchanged and list it. Unchanged is always acceptable.

### 4.1 Tone for sensitive domains

| Domain | Tone |
|---|---|
| `loss` (pregnancy loss) | Gentle, plain, no cheer, no «!», no clichés («متأسفانه»، «قوی باش»، «خدا بزرگ است»), never blame. Short sentences. Offer, don't instruct («اگر خواستی …»). |
| `ivf` | Calm, factual, never promise outcomes («موفقیت» only where the text already reports a result). |
| `menopause` | Respectful adult tone, no «پیری»-framing, symptoms named plainly. |
| `teen` | Same «تو», simpler words, shorter sentences, no fertility/pregnancy wording (contract `no_fertility_copy`). |
| `conditions`, `labs`, `vitals` alerts | Clear and calm; urgency exactly as written. |
| Privacy / delete account / consent | Precise, complete, no softening of what is deleted or shared. |

---

## 5. Banned (AI / bureaucratic tells)

می‌باشد، محسوب می‌شود، به شمار می‌رود، گردید، لازم به ذکر است، شایان ذکر است، در راستای، از اهمیت ویژه‌ای
برخوردار است، نقش بسزایی، بی‌نظیر، فوق‌العاده (in promises), منحصربه‌فرد، تجربه‌ای متفاوت، نه تنها … بلکه،
«که نشان‌دهندهٔ … است» tails, «در دنیای امروز»، «در نهایت می‌توان گفت»، rule-of-three triads in
marketing copy, em dashes, «!!»، emoji in copy (keep an emoji only if the UI design depends on it — list it).

---

## 6. ICU / JSON safety (enforced by the checker)

- Never change keys, key order, `{arg}` names, `{n, plural, …}` / `{x, select, …}` structure, arm names
  (`one`, `other`, `=0`, `female` …), `#`, or `<tag>…</tag>` names. Words inside arms may change.
- Persian has one plural form: `{n, plural, other{# روز}}` is enough; keep any `=0`/`one` arm that exists.
- Never build sentences from fragments; if a string is a fragment by design (e.g. used inside another),
  keep its grammatical shape.
- `en` is the meaning reference, never edited.
- Never run `persian_cleanup.py` on a raw JSON file; use `frontend/scripts/copy-fa/clean.mjs`.

---

## 7. Before → after (real keys)

| Key | Before | After |
|---|---|---|
| `home.title` | به ریتمی خوش اومدی | به ریتمی خوش آمدی |
| `home.updating` | داریم سیکلت رو به‌روز می‌کنیم… | در حال به‌روزرسانی چرخه‌ات… |
| `home.phaseDescription.period` | روزهای قاعدگی؛ بدنت پوشش داخلی رحم رو دفع می‌کنه و انرژی معمولاً پایین‌تره. | روزهای پریود؛ بدنت پوشش داخلی رحم را دفع می‌کند و انرژی معمولاً پایین‌تر است. |
| `home.phaseDescription.follicular` | بعد از قاعدگی، بدنت برای تخمک‌گذاری آماده می‌شه و انرژی‌ات کم‌کم بالا می‌ره. | بعد از پریود، بدنت برای تخمک‌گذاری آماده می‌شود و انرژی‌ات کم‌کم بالا می‌رود. |
| `pwa.softTitle` | یه نسخه جدید اومده | نسخهٔ تازه‌ای آماده است |
| `pwa.forcedBody` | این نسخه دیگه پشتیبانی نمی‌شه. برای این‌که بتونی از ریتمی استفاده کنی، به‌روزرسانی کن. | این نسخه دیگر پشتیبانی نمی‌شود. برای ادامهٔ استفاده از ریتمی، به‌روزرسانی کن. |
| `pwa.installBody` | ریتمی رو روی صفحه اصلی گوشیت بذار تا سریع‌تر بهش برسی. | ریتمی را به صفحهٔ اصلی گوشی‌ات اضافه کن تا سریع‌تر به آن برسی. |
| `pwa.iosStep1` | ۱. توی نوار پایین سافاری، دکمه اشتراک‌گذاری رو بزن | ۱. در نوار پایین سافاری، دکمهٔ اشتراک‌گذاری را بزن |
| `pregnancy-v2.common.loadError` | اطلاعات بارداری نیومد. یه‌کم بعد دوباره امتحان کن. | اطلاعات بارداری بارگیری نشد. کمی بعد دوباره امتحان کن. |
| `pregnancy.error` | یه مشکلی پیش اومد؛ دوباره امتحان کن. | مشکلی پیش آمد. دوباره امتحان کن. |
| `pregnancy.notActive.body` | با روشن‌کردن حالت بارداری، هفته‌به‌هفته رشد کودکت و تغییرات بدنت رو دنبال می‌کنی و مراقبت‌های هر هفته رو می‌بینی. | با روشن کردن حالت بارداری، رشد کودکت و تغییرات بدنت را هفته‌به‌هفته دنبال می‌کنی و مراقبت‌های هر هفته را می‌بینی. |
| `calendar.subtitle` | روزهای دوره و پیش‌بینی‌هات رو این‌جا ببین. | روزهای پریود و پیش‌بینی‌هایت را این‌جا ببین. |
| `calendar.jumpToMonth` | رفتن به یه ماه دیگه | رفتن به ماهی دیگر |
| `fertility.log.futureDate` | برای روزهای آینده نمی‌شه ثبت کرد. | برای روزهای آینده نمی‌توانی ثبت کنی. |
| `fertility.log.tooManyRequests` | یه لحظه صبر کن و دوباره ذخیره کن؛ توی این یک دقیقه تغییرهای زیادی فرستاده شده. | چند لحظه صبر کن و دوباره ذخیره کن؛ در یک دقیقهٔ گذشته تغییرهای زیادی فرستاده شده است. |
| `account.delete.warning` | با حذف حساب، همهٔ اطلاعاتت — پروفایل، سوابق دوره، … — برای همیشه پاک می‌شه و دیگه نمی‌شه برشون گردوند. | با حذف حساب، همهٔ اطلاعاتت (پروفایل، سوابق پریود، …) برای همیشه پاک می‌شود و دیگر نمی‌توان آن‌ها را برگرداند. |
| `log-sheet.pain.map` | نقشه بدن — جای درد را انتخاب کن | نقشهٔ بدن: جای درد را انتخاب کن |
| `log-sheet.measure.empty` | ثبت نشده — با + یا − شروع کن | هنوز ثبت نشده است. با + یا − شروع کن. |
| `me.notifSettings.quiet.off` | خاموش — اعلان‌ها هر ساعتی می‌رسند | خاموش؛ اعلان‌ها در هر ساعتی می‌رسند |
| `ivf.meds.form.trackStockHint` | وقتی دارو رو به اتمام است «کم است» نشان می‌دهیم. | وقتی دارو رو به اتمام باشد، برچسب «کم است» را نشان می‌دهیم. |

