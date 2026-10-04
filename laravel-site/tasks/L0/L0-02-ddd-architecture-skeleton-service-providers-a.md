---
id: L0-02
title: DDD architecture skeleton, service providers and arch tests
milestone: L0
type: backend
status: done
depends_on: [L0-01]
parallel_group: L0-A
touches: [app/Domain,app/Support,app/Providers,tests/Arch,docs/ARCHITECTURE.md]
skills: []
verify: composer verify
---

# L0-02 — DDD architecture skeleton, service providers and arch tests

## Why
A predictable structure keeps 60+ tasks consistent and makes SOLID/KISS enforceable by tests, not by memory.

## Scope
- Bounded contexts (empty folders + `README.md` one-liners only where code lands later, YAGNI otherwise):
  `Seo`, `Settings`, `Media`, `Content` (static pages registry), `Faq`, `Contact`, `Blog`, `Newsletter`,
  `Directory`, `Shop`, `Pwa`. Each context: `Models/`, `Actions/` (single-purpose invokable use cases),
  `Data/` (readonly DTOs), `Contracts/` (repository + service interfaces), `Repositories/` (Eloquent + `Cached*`
  decorators), `Events/`, `Enums/`, `Observers/` — created only when first needed.
- `app/Support`: shared kernel (Cache, Html, Jalali, Money, Text helpers) — framework-light.
- `DomainServiceProvider` base + one provider per context registering bindings (interface → cached decorator →
  Eloquent impl), observers, and view composers. Registered in `bootstrap/providers.php`.
- `docs/ARCHITECTURE.md` (≤ 1 page): layers, dependency rule (Http/Filament → Actions/Contracts → Models), naming,
  where queries live (repositories / query objects; never in Blade or controllers), DTOs into views (no lazy loading in
  views: `Model::preventLazyLoading()` in non-production), how to add a context.
- Pest arch tests: `App\Domain` does not use `Illuminate\Http`, `App\Http`, `App\Filament`; controllers are final and
  don't call `DB::`/Eloquent builders directly; no `env()` outside `config/`; no `dd/dump/ray`; strict types declared;
  Actions are invokable/`handle()` and final.

## Out of scope
- Concrete domain models (each context task owns its own).

## Acceptance
- Arch tests run inside `composer verify` and pass; `Model::preventLazyLoading/preventSilentlyDiscardingAttributes`
  on outside production; `docs/ARCHITECTURE.md` exists.
