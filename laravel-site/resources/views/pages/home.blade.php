{{--
    Home page, Nov 2026 design (design/Main.html). Data from App\Http\Controllers\HomeController:
      $heroPhoto MediaData|null, $heroAlt, $appLinks AppLinksSettings — the per-request hero (LCP photo, priority)
      $staticSections HtmlString — pages/home/static.blade.php pre-rendered and cached (`pages` namespace)
      $readings       list<PostCardData> — six newest posts; hidden while the magazine is empty / switched off
      $faqItems       list<array{question, answer}>, $faqTitle — FAQ group `home` ($faq from the FaqServiceProvider composer)
    Copy: lang/fa/home.php. Order: hero → stages → #features → steps → partner → privacy → readings → #download → FAQ.
    The header is the light variant (StaticPage::headerVariant), so the hero is plain content.
--}}
@extends('layouts.app')

@section('content')
@php
    $faqItems = array_map(static fn ($item): array => ['question' => $item->question, 'answer' => $item->answer], ($faq ?? null)?->items ?? []);
@endphp
<div class="bleed-rows bg-surface">
    @include('pages.home.hero')

    {{ $staticSections }}

    @siteSection('blog')
    <x-site.readings
        :posts="$readings"
        :eyebrow="__('home.readings.eyebrow')"
        :title="__('home.readings.title')"
        :more-href="route('blog.index')"
        :more-label="__('home.readings.more')"
        id="home-readings"
    />
    @endsiteSection

    <x-site.download-band :title="__('home.download.title')" :lead="__('home.download.lead')" :links="$appLinks"/>

    <x-site.faq :items="$faqItems" :title="($faq ?? null)?->title ?? __('home.faq.title')" id="home-faq"/>
</div>
@endsection
