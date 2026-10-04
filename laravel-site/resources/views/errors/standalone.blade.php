{{--
    Standalone status page for 500 and 503: no layout components, settings, SeoManager or cache, because the
    database or the cache may be the reason for the error (or be migrating in maintenance mode). Only the compiled
    CSS from the Vite manifest (skipped when there is no build). Data: $code, $title (the single h1), $message.
--}}
@php
    try {
        $standaloneCss = app(\Illuminate\Foundation\Vite::class)(['resources/css/app.css'])->toHtml();
    } catch (\Throwable) {
        $standaloneCss = '';
    }
@endphp
<!doctype html>
<html lang="fa" dir="rtl">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>{{ $title }} — ریتمی</title>
    <meta name="robots" content="noindex, nofollow">
    {!! $standaloneCss !!}
</head>
<body class="antialiased">
    <main class="grid min-h-screen place-items-center bg-canvas px-5 py-16">
        <div class="w-full max-w-[560px] text-center">
            <a href="{{ url('/') }}" class="font-display text-d-sm leading-none text-primary">ریتمی</a>
            <p class="mt-10 font-display text-d-md leading-none text-muted">{{ strtr((string) $code, ['0' => '۰', '1' => '۱', '2' => '۲', '3' => '۳', '4' => '۴', '5' => '۵', '6' => '۶', '7' => '۷', '8' => '۸', '9' => '۹']) }}</p>
            <h1 class="mt-4 font-display text-d-lg leading-display text-ink">{{ $title }}</h1>
            <p class="mt-4 text-xl leading-relaxed text-muted">{{ $message }}</p>
            <a href="{{ url('/') }}" class="mt-8 inline-flex h-13 items-center rounded-full bg-primary px-6 text-md font-extrabold text-white hover:bg-primary-hover hover:text-white">صفحه اصلی</a>
        </div>
    </main>
</body>
</html>
