# App\Support — shared kernel

Framework-light helpers every context may use. Nothing here depends on `App\Domain`, `App\Http` or `App\Filament`.
Modules are added by the task that first needs them (YAGNI):

| Namespace | Purpose | Owner task |
|---|---|---|
| `App\Support\Cache` | `CacheAside` + versioned namespaces for cached repository decorators | L0-03 |
| `App\Support\Html` | rich-HTML sanitiser, external links, Persian heading anchors/TOC, word count + reading time, slugs | L4-01 |
| `App\Support\Jalali` | `JalaliCalendar` (exact Jalali ⇄ Gregorian, leap years; jalaali-js port, no package) + `JalaliDate` (Tehran-time value, `format()` tokens, Persian digits); global `jdate()` | L3-01b |
| `App\Support\Money` | Toman/Rial value object + Persian formatting | L6 |
| `App\Support\Text` | `PersianDigits` (to/from Persian digits, «٬»/«٫» number format), `Toman` (minimal display formatter); global `fa_digits()` | L3-01b |

Global helpers live in `app/Support/helpers.php` (composer `autoload.files`) and only wrap the classes above:
`fa_digits($value)` and `jdate($date = null, $format = 'j F Y', $persianDigits = true)`. Admin tables should use
`jdate()` for dates (Filament `->formatStateUsing(fn ($state) => $state ? jdate($state, 'Y/m/d H:i') : null)`).
