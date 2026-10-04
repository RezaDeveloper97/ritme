{{--
    Home page (L3-02), design/html/index.html. Data from App\Http\Controllers\HomeController:
      $staticSections HtmlString — pages/home/static.blade.php pre-rendered and cached (`pages` namespace)
      $readings       list<PostCardData> — newest posts; the block renders nothing while the magazine is empty
      $appLinks       AppLinksSettings, $qrUrl string — the #download card
    Copy: lang/fa/home.php. Sections: hero (dark block, the LCP is the h1 — text, nothing lazy) → static sections →
    readings → FAQ «قبل از نصب» (L3-09 slot) → #download → promise banner.
--}}
@extends('layouts.app')

@section('hero')
    @include('pages.home.hero')
@endsection

@section('content')
    {{ $staticSections }}

    <x-stage.readings
        :posts="$readings"
        :eyebrow="__('home.readings.eyebrow')"
        :title="__('home.readings.title')"
        :more-href="route('blog.index')"
        :more-label="__('home.readings.more')"
        id="home-readings"
        bg="surface"
    />

    {{-- ===== L3-09 SLOT: FAQ «قبل از نصب» (design: eyebrow «سؤال‌های رایج», centred h2, 4 items, first open) =====
         L3-09 owns the FAQ data (group `home`) and its FAQPage JSON-LD. Replace this comment with its block, e.g.
         <x-faq group="home" eyebrow="سؤال‌های رایج" title="قبل از نصب" align="center"/>
         Until then nothing renders here: no hard-coded FAQ copy and no empty shell. ===== --}}

    <x-ui.app-cta
        :title="__('home.app_cta.title')"
        :lead="__('home.app_cta.lead')"
        :links="$appLinks"
        :qr-url="$qrUrl"
    />

    <x-ui.promise-banner :href="route('social-responsibility')"/>
@endsection
