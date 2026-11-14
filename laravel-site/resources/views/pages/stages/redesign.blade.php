{{--
    Nov 2026 stage-page design (cycle, ttc, pregnancy; postpartum/menopause/teen stay on show.blade.php). $page is a
    StagePageData with `redesign` set. Sections: tinted hero band (text + photo with two floating chips) →
    #features (photo + six cards) → tools list on canvas → emergency → readings → FAQ → #download band.
    Colour utilities are literal here on purpose (the class scanner does not read app/Domain).
--}}
@extends('layouts.app')

@php
    $r = $page->redesign;
    [$text, $dash, $band] = match ($r->accent) {
        'ttc' => ['text-stage-ttc', 'bg-stage-ttc', 'bg-sand'],
        'pregnancy' => ['text-stage-pregnancy-warm', 'bg-stage-pregnancy-warm', 'bg-peach'],
        default => ['text-stage-cycle', 'bg-stage-cycle', 'bg-danger-soft'],
    };
@endphp

@section('hero')
    <section aria-labelledby="stage-title" class="bg-surface px-6 pt-6 max-sm:px-3 max-sm:pt-3">
        <div class="relative overflow-hidden rounded-6xl {{ $band }} max-sm:rounded-4xl">
            <div class="relative flex items-center gap-18 px-24 py-18 max-xl:px-10 max-lg:flex-col max-lg:items-stretch max-lg:gap-8 max-lg:py-12 max-sm:gap-5 max-sm:px-5 max-sm:pt-8 max-sm:pb-5">
                <div class="flex flex-1 flex-col gap-6 max-sm:gap-5">
                    <span class="text-base font-extrabold {{ $text }}">{{ $r->eyebrow }}</span>
                    <h1 id="stage-title" class="m-0 font-display text-[60px] leading-[1.2] font-normal text-ink max-lg:text-[44px] max-sm:text-[38px] max-sm:leading-[1.25]">{{ $r->title }}</h1>
                    <p class="m-0 max-w-135 text-xl leading-loose font-medium text-muted max-sm:text-lg">{{ $r->lead }}</p>
                    <div class="flex flex-wrap items-center gap-7 max-sm:flex-col max-sm:items-stretch max-sm:gap-4 max-sm:[&>a:last-child]:self-center">
                        <x-site.button href="#download" icon="download">{{ $r->copy['download'] }}</x-site.button>
                        <x-site.text-link href="#features">{{ $r->copy['see'] }}</x-site.text-link>
                    </div>
                </div>
                <div class="relative w-82.5 shrink-0 ms-10 max-lg:mx-auto max-lg:ms-0 max-sm:w-full">
                    <span class="relative block h-110 overflow-hidden rounded-4xl bg-canvas max-sm:h-105">
                        @if ($r->heroPhoto)
                            <x-picture :media="$r->heroPhoto" :alt="$r->heroAlt" sizes="(min-width: 1024px) 330px, 100vw" priority class="size-full object-cover"/>
                        @endif
                    </span>
                    <div class="absolute top-16 -start-7 flex flex-col gap-0.5 rounded-xl bg-surface px-4 py-3 shadow-pop max-sm:hidden">
                        <b class="text-base text-ink">{{ $r->chips[0]['title'] }}</b>
                        <span class="text-[12.5px] font-semibold text-muted">{{ $r->chips[0]['text'] }}</span>
                    </div>
                    <div class="absolute bottom-18 -end-7 flex flex-col gap-0.5 rounded-xl bg-surface px-4 py-3 shadow-pop max-sm:end-auto max-sm:start-4 max-sm:bottom-4">
                        <b class="text-base text-ink">{{ $r->chips[1]['title'] }}</b>
                        <span class="text-[12.5px] font-semibold text-muted">{{ $r->chips[1]['text'] }}</span>
                    </div>
                </div>
            </div>
        </div>
    </section>
@endsection

@section('content')
<div class="bleed-rows bg-surface">
    <div id="features" class="scroll-mt-6">
        <section aria-labelledby="stage-features-title" class="flex flex-col gap-10 bg-surface px-30 py-24 max-xl:px-6 max-lg:gap-6 max-lg:px-5 max-lg:py-14">
            <x-site.heading id="stage-features-title" :eyebrow="$r->copy['featuresEyebrow']" :eyebrow-color="$text" :title="$r->featuresTitle"/>
            <div class="flex gap-6 max-lg:flex-col max-lg:gap-3">
                <div class="w-100 shrink-0 self-stretch overflow-hidden rounded-4xl bg-canvas max-lg:h-50 max-lg:w-full max-lg:rounded-3xl lg:h-100 lg:self-auto">
                    @if ($r->featurePhoto)
                        <x-picture :media="$r->featurePhoto" :alt="$r->featureAlt" sizes="(min-width: 1024px) 400px, 100vw" class="size-full object-cover"/>
                    @endif
                </div>
                <ul class="m-0 grid flex-1 list-none grid-cols-2 gap-4 p-0 max-lg:gap-2.5">
                    @foreach ($r->features as $feature)
                        <li class="flex flex-col gap-2 rounded-3xl border border-line bg-surface p-6 max-sm:gap-1.5 max-sm:rounded-xl max-sm:px-3.5 max-sm:py-4">
                            <span class="h-[3px] w-6 rounded-sm {{ $dash }} max-sm:w-5" aria-hidden="true"></span>
                            <b class="text-2xl leading-relaxed max-sm:text-md">{{ $feature['title'] }}</b>
                            <span class="text-md leading-[1.8] font-medium text-muted max-sm:text-[13px]">{{ $feature['text'] }}</span>
                        </li>
                    @endforeach
                </ul>
            </div>
        </section>
    </div>

    @if ($page->tools !== [])
        <section aria-labelledby="stage-tools-title" class="flex flex-col gap-10 bg-canvas px-30 py-24 max-xl:px-6 max-lg:gap-6 max-lg:px-5 max-lg:py-14">
            <x-site.heading id="stage-tools-title" :eyebrow="$r->copy['toolsEyebrow']" :title="$r->copy['toolsTitle']"/>
            <ul class="m-0 grid list-none grid-cols-3 gap-x-10 p-0 max-lg:grid-cols-2 max-sm:grid-cols-1">
                @foreach ($page->tools as $tool)
                    <li>
                        <a href="{{ $tool->href }}" class="flex items-start justify-between gap-3 border-t border-line py-5.5 text-ink">
                            <span class="flex flex-col gap-1">
                                <b class="text-xl">{{ $tool->title }}</b>
                                @if ($tool->text)<span class="text-md leading-[1.8] font-medium text-muted">{{ $tool->text }}</span>@endif
                            </span>
                            @if ($tool->where)
                                <span @class(['shrink-0 rounded-lg px-2.5 py-1 text-[12.5px] font-extrabold', 'bg-success-soft text-success' => $tool->onSite, 'bg-canvas text-muted' => ! $tool->onSite])>{{ $tool->where }}</span>
                            @endif
                        </a>
                    </li>
                @endforeach
            </ul>
        </section>
    @endif

    <x-site.readings :posts="$page->readings" :more-href="$page->readingsMoreUrl" :more-label="$r->copy['readingsMore']" id="stage-readings"/>

    <div data-faq-group="{{ $page->faqGroup }}">
        <x-site.faq :items="$page->faqProps()" :title="$r->copy['faqTitle']" id="stage-faq"/>
    </div>

    <x-site.download-band :title="$r->copy['downloadTitle']" :lead="$r->copy['downloadLead']" :links="$page->appLinks"/>
</div>
@endsection
