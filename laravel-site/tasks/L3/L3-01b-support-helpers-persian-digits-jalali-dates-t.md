---
id: L3-01b
title: Support helpers: Persian digits, Jalali dates, Toman formatting
milestone: L3
type: backend
status: todo
depends_on: [L3-01]
parallel_group: L3-A
touches: [app/Support/Text,app/Support/Jalali,app/Support/helpers.php,composer.json,resources/views/components/ui,resources/views/components/cards,app/View/Components/Layout,tests/Unit/Support]
skills: []
verify: composer verify
---

# L3-01b — Support helpers: Persian digits, Jalali dates, Toman formatting

## Why
L3-01 shipped the component kit without the shared helpers its scope asked for (`app/Support` was outside its touches).
Components currently borrow `App\View\Components\Layout\Footer::persianDigits()` and an inline Toman formatter, and
dates arrive as pre-formatted strings. Filament tables (L1-08) also show Gregorian dates.

## Scope
- `App\Support\Text\PersianDigits` (to/from Persian digits, Persian separators) + global `fa_digits()` helper via
  composer `autoload.files` (`app/Support/helpers.php`).
- `App\Support\Jalali` date formatting (use a maintained package such as `morilog/jalali` if it supports PHP 8.2 —
  else a small converter) + `jdate()` helper; tested against known dates.
- Minimal Toman formatter in `App\Support\Text` (full `Money` value object stays with L6-01).
- Replace `Footer::persianDigits()` and the inline formatter in `x-ui.price` / `x-ui.rating` etc. with the helpers.

## Out of scope
- Filament table date columns (later admin tasks use `jdate()`), Money value object (L6-01).

## Acceptance
- Unit tests for digits, Jalali conversions (incl. leap years) and Toman formatting; components use the helpers.
- `composer verify` green.
