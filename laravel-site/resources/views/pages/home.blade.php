{{--
    Home page (L3-02), design/html/index.html. Data from App\Http\Controllers\HomeController:
      $staticSections HtmlString — pages/home/static.blade.php pre-rendered and cached (`pages` namespace)
      $readings       list<PostCardData> — newest posts; the block renders nothing while the magazine is empty
      $appLinks       AppLinksSettings, $qrUrl string — the #download card
      $faq            FaqGroupData|null — FAQ group `home`, from the FaqServiceProvider view composer (L3-09)
    Copy: lang/fa/home.php. Sections: hero (dark block, the LCP is the h1 — text, nothing lazy) → static sections →
    readings → FAQ «قبل از نصب» (x-faq, group `home`) → #download → promise banner.
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

    {{-- FAQ «قبل از نصب» (L3-09): group `home` from the FaqServiceProvider composer ($faq, null → nothing renders);
         its questions are already in the page's FAQPage JSON-LD. The h2 is the group title (admin-editable). --}}
    <x-faq :faq="$faq ?? null" eyebrow="سؤال‌های رایج" :title="($faq ?? null)?->title" align="center" heading-id="home-faq-title"/>

    <x-ui.app-cta
        :title="__('home.app_cta.title')"
        :lead="__('home.app_cta.lead')"
        :links="$appLinks"
        :qr-url="$qrUrl"
    />

    <x-ui.promise-banner :href="route('social-responsibility')"/>
@endsection
