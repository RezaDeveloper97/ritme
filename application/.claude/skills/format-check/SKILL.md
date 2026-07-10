---
name: format-check
description: Check and fix Kotlin formatting/style for the Ritme project — indentation, imports, immutability, magic numbers/colors, naming, KDoc. Use after finishing a feature or when the user asks to clean up/format code.
---

# Format & Style Check — Ritme

For many files, delegate to the `kotlin-style-fixer` subagent (it may edit). For one or two files, fix inline. Style only — never change behavior.

## Rules (from CLAUDE.md §4, §5c)

- 4-space indent, no tabs, no trailing whitespace, newline at EOF, no wildcard imports, ~120 char lines.
- `val` over `var`; immutable data classes for domain models; expose read-only collection types.
- No magic numbers/strings — extract `private const val`. No inline `Color(0xFF...)` in Composables — use `RitmeTheme`/`colors.xml` tokens.
- Naming: use cases as verbs (`LogPeriodUseCase`), repositories as nouns; no abbreviations except `vm`/`id`.
- Sealed classes for finite states instead of nullable/boolean flags (report, don't auto-refactor).
- Short "why" KDoc on public functions in `domain/` and `application/`.
- One class per file; no `Utils.kt`.

## After fixing

Run `./gradlew --offline compileDebugKotlin` (dl.google.com is blocked for new deps; offline works with warm cache) to confirm nothing broke, then list files changed and any deferred behavioral issues.
