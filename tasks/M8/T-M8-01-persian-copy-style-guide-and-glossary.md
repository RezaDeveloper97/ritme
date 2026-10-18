---
id: T-M8-01
title: Persian copy style guide, glossary and namespace inventory
milestone: M8
type: investigate
status: done
depends_on: []
parallel_group: M8-A
touches: [docs/copy-fa/STYLE.md,docs/copy-fa/GLOSSARY.md,docs/copy-fa/INVENTORY.md]
skills: [persian-writing:persian-writing]
verify: test -s docs/copy-fa/STYLE.md && test -s docs/copy-fa/GLOSSARY.md && test -s docs/copy-fa/INVENTORY.md
---

# T-M8-01 — Persian copy style guide, glossary and namespace inventory
**Model:** main session (opus). This is the contract every rewrite batch follows — do not delegate.

## Why
The user wants every Persian string of the user app (`frontend/messages/fa/*.json`, 70 namespaces, ~560 KB)
rewritten with the `persian-writing` skill. Today the register is mixed: `home` is colloquial («خوش اومدی»،
«می‌کنه»), `auth` mixes written («ریتم بدنت را بشناس») and colloquial («یه مشکلی پیش اومد»). Rewriters (sonnet
subagents) need one fixed contract or every batch drifts.

**User decision (2026-10-18): register = written-but-warm, addressing the reader as «تو».**
Full written verb forms (می‌شود، آمدی، را), second-person singular (بدنت، می‌توانی), no slang, no «می‌باشد».
Scope = user app only (`frontend`). admin-web, instructor-web, backend-go copy and laravel-site are out of scope.

## Scope
1. `docs/copy-fa/STYLE.md` (English prose, Persian examples): register rules per string kind (button/CTA,
   title, empty state, error, toast, health explanation, privacy/consent/legal, onboarding), length limits for
   buttons/tabs/chips (keep ≤ the current length unless the old one is wrong), «تو» everywhere, POV rule («ما» only
   for what Ritmi does), orthography (ZWNJ, ی/ک, «گیومه», ، ؛ ؟, no em dash, ellipsis «…»), digits policy for
   JSON (literal digits in prose → Persian; `{n, number}` placeholders stay as-is), ICU rules (placeholders,
   plural/select arms, rich-text tags are untouchable), medical-safety rules (never change a number, threshold,
   dose, time window, or certainty level; keep every hedge «معمولاً/ممکن است»; never add a claim), tone for
   sensitive domains (loss, IVF, menopause, teen). Include 15–20 before/after pairs drawn from real keys.
2. `docs/copy-fa/GLOSSARY.md`: one canonical term per concept with the rejected variants, e.g.
   قاعدگی/پریود، چرخه/سیکل، پنجره‌ی باروری، تخمک‌گذاری، فاز لوتئال/فولیکولار, ثبت/لاگ, یادآور, پروفایل,
   برنامه‌ی پلاس … Build it from a frequency count over `frontend/messages/fa` (show the counts), pick the
   winner by what Iranian women's-health apps and gynaecologists' patient material use; brand name «ریتمی».
3. `docs/copy-fa/INVENTORY.md`: per namespace — size, key count, current register mix (written/colloquial
   ratio), sensitivity (medical / emotional / commercial / UI), and the batch task it belongs to (T-M8-03…13).
4. Show the user the STYLE.md summary + 10 before/after pairs + the glossary decisions that are not obvious, in
   Persian, and **wait for their OK** before setting this task done. Record the approval date in STYLE.md.

## Out of scope
- Editing any message file.

## Acceptance
- The three docs exist; the user approved STYLE.md + GLOSSARY.md in chat.
- Every one of the 70 namespaces appears in INVENTORY.md with a batch id.
