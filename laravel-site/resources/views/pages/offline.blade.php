{{--
    /offline (L8-02): precached by the service worker and shown for a navigation that fails without a cached copy
    (so it is rendered at the URL the visitor tried — «تلاش دوباره» reloads that URL). Static, noindex (set in
    OfflineController). The retry button stays hidden until the `pwa` module wires it; the home link works without JS.
--}}
@extends('layouts.app', ['appCta' => false])

@section('content')
    <x-ui.section pad="lg">
        <div class="flex max-w-170 flex-col gap-4">
            <span class="inline-flex size-16 items-center justify-center rounded-3xl bg-surface text-primary">
                <x-icon name="info" class="size-8"/>
            </span>
            <h1 class="m-0 font-display text-d-xl leading-display text-ink">اتصال اینترنت برقرار نیست</h1>
            <p class="m-0 text-xl leading-relaxed text-muted">
                به نظر می‌رسد گوشی یا رایانه‌ات الان به اینترنت وصل نیست. صفحه‌هایی که قبلاً در ریتمی باز کرده‌ای بدون
                اینترنت هم در دسترس‌اند؛ برای بقیه، وقتی اتصال برگشت دوباره تلاش کن.
            </p>
            <div class="mt-4 flex flex-wrap items-center gap-3">
                <x-ui.button size="lg" icon="return" data-pwa-retry hidden>تلاش دوباره</x-ui.button>
                <x-ui.button href="/" variant="outline" size="lg" icon-end="arrow-left">صفحه اصلی</x-ui.button>
            </div>
        </div>
    </x-ui.section>
@endsection
