# Architecture (DDD-lite)

## Layers and the dependency rule

```
app/Http, app/Filament   (delivery: controllers, requests, view composers, Filament resources — thin)
        │ call
        ▼
app/Domain/<Context>/Actions, Contracts   (use cases + repository/service interfaces)
        │ implemented by / operate on
        ▼
app/Domain/<Context>/Repositories, Models (Eloquent)        app/Support (shared kernel, used by every layer)
```

Dependencies only point down. `App\Domain` never uses `Illuminate\Http`, `App\Http` or `App\Filament`; `App\Support`
never uses `App\Domain`/`App\Http`/`App\Filament`. Enforced by `tests/Arch` (runs in `composer verify`).

Contexts: `Seo`, `Settings`, `Media`, `Content` (static pages registry), `Faq`, `Contact`, `Blog`, `Newsletter`,
`Directory`, `Shop`, `Pwa`.

## Inside a context (`app/Domain/<Context>/`) — folders are created only when first needed

| Folder | Holds | Naming |
|---|---|---|
| `Models/` | Eloquent models | `Post` |
| `Actions/` | one use case each, `final`, `__invoke()` or `handle()` | `PublishPost` |
| `Data/` | `final readonly` DTOs handed to views/APIs | `PostCardData` |
| `Contracts/` | repository + service interfaces | `PostRepository` |
| `Repositories/` | `Eloquent*` impl + `Cached*` decorator (cache-aside via `App\Support\Cache`) | `EloquentPostRepository`, `CachedPostRepository` |
| `Events/`, `Enums/`, `Observers/` | domain events, backed enums, model observers (bump cache namespaces) | `PostPublished`, `PostStatus`, `PostObserver` |

## Where queries live

Only in repositories or dedicated query objects. **Never** in controllers, Blade, view composers or Filament page
code beyond Filament's own resource queries. Controllers are `final`, may not use `DB`/query builders (arch test), and
pass **DTOs** to views. `Model::preventLazyLoading()` and `preventSilentlyDiscardingAttributes()` are on outside
production (`AppServiceProvider`), so a view that triggers a lazy load fails in dev and tests.

## Wiring

Each context has `App\Providers\Domain\<Context>ServiceProvider extends App\Providers\DomainServiceProvider`, listed in
`bootstrap/providers.php`. It only declares arrays:

```php
protected array $repositories = [PostRepository::class => [EloquentPostRepository::class, CachedPostRepository::class]];
protected array $observers    = [Post::class => PostObserver::class];
protected array $composers    = ['partials.footer' => FooterComposer::class]; // composers live in App\Http\View
```

A repository contract resolves (singleton) to the cached decorator, which receives the Eloquent implementation through
its constructor parameter **`$inner`**. Plain services use Laravel's `$bindings` / `$singletons`.

## General rules

`declare(strict_types=1)` everywhere; `env()` only in `config/`; no `dd/dump/ray`; PHP 8.2-compatible syntax; SOLID,
KISS, YAGNI — no folder, interface or decorator before a task needs it.

## Adding a context

1. `app/Domain/<Context>/README.md` (one line of purpose) and the folders you actually need.
2. `app/Providers/Domain/<Context>ServiceProvider.php` (`final`, extends `DomainServiceProvider`), add it to
   `bootstrap/providers.php`.
3. Contract → `Eloquent*` → `Cached*`, register in `$repositories`; observer bumps the cache namespaces it affects
   (incl. `pages`).
4. Add the context name to `tests/Arch/DomainServiceProviderTest.php`.
