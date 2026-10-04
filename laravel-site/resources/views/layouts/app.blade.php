{{--
    Public page shell. Pages: @extends('layouts.app') with
      @section('hero')    optional — rendered inside the dark header block on dark-header pages (AUDIT §2.1)
      @section('content') the page body (first child of <main id="main">)
    Optional view data: $headerVariant ('dark'|'light'), $navRoute (route name used for the active nav item),
    $appCta (bool: the page renders the #download app CTA). Defaults come from the StaticPage registry.
    SEO tags come from <x-seo.head/>; breadcrumbs (<x-ui.breadcrumbs/>) belong in a section so they register
    their JSON-LD before the head renders. Scripts: lazy data-module ES modules via app.js; @stack('scripts') last.
    body[data-module=pwa]: service worker, two-tier update toast / forced screen, install prompt (L8-02).

    Dark pages: header, mobile menu and hero share ONE night/glow background (a grid layer under rows 1–3; the
    hero row reaches the layer through `grid-rows-subgrid` on <main>, so the hero stays inside the main landmark).
    Browsers without subgrid give the hero its own background instead.
--}}
@php
    $layoutRoute = $navRoute ?? \Illuminate\Support\Facades\Route::currentRouteName();
    $layoutPage = \App\Domain\Content\Enums\StaticPage::forRoute($layoutRoute);
    $layoutVariant = \App\Domain\Content\Enums\HeaderVariant::resolve(
        $headerVariant ?? null,
        $layoutPage?->headerVariant() ?? \App\Domain\Content\Enums\HeaderVariant::Light,
    );
    $layoutHasHero = trim($__env->yieldContent('hero')) !== '';
@endphp
<!doctype html>
<html lang="fa" dir="rtl">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <x-layout.assets/>
    <x-seo.head/>
    @stack('head')
</head>
<body class="antialiased" data-module="pwa">
    <a href="#main" class="sr-only focus:not-sr-only focus:fixed focus:start-4 focus:top-4 focus:z-50 focus:rounded-full focus:bg-primary focus:px-5 focus:py-3 focus:text-base focus:font-extrabold focus:text-white">رفتن به محتوای اصلی</a>
    <div class="relative mx-auto grid w-full max-w-page grid-cols-1 overflow-hidden bg-canvas">
        @if ($layoutVariant->isDark())
            <div aria-hidden="true" @class([
                'col-start-1 row-start-1 row-end-3 bg-night bg-hero-glow',
                'supports-[grid-template-rows:subgrid]:row-end-4' => $layoutHasHero,
            ])></div>
        @endif
        <x-layout.header :variant="$layoutVariant" :route="$layoutRoute" :app-cta="$appCta ?? null"/>
        <main id="main" tabindex="-1" class="col-start-1 row-start-3 row-end-5 grid grid-cols-1 grid-rows-subgrid focus:outline-none">
            @if ($layoutHasHero)
                <div @class([
                    'relative',
                    'not-supports-[grid-template-rows:subgrid]:bg-night not-supports-[grid-template-rows:subgrid]:bg-hero-glow' => $layoutVariant->isDark(),
                ])>
                    @yield('hero')
                </div>
            @endif
            <div>
                @yield('content')
            </div>
        </main>
        <x-layout.footer/>
    </div>
    @stack('scripts')
</body>
</html>
