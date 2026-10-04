# App\Support — shared kernel

Framework-light helpers every context may use. Nothing here depends on `App\Domain`, `App\Http` or `App\Filament`.
Modules are added by the task that first needs them (YAGNI):

| Namespace | Purpose | Owner task |
|---|---|---|
| `App\Support\Cache` | `CacheAside` + versioned namespaces for cached repository decorators | L0-03 |
| `App\Support\Html` | small HTML/attribute helpers for Blade components | first user |
| `App\Support\Jalali` | Jalali (Shamsi) date formatting / parsing | first user |
| `App\Support\Money` | Toman/Rial value object + Persian formatting | L6 |
| `App\Support\Text` | Persian text helpers (digits, slugs, excerpts) | first user |
