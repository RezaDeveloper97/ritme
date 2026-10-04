{{--
    Privacy page (L3-08), design/html/privacy.html. Data from App\Http\Controllers\PrivacyController:
      $updatedAt  ?CarbonImmutable — «آخرین به‌روزرسانی» (also the page's dateModified)
      $dpoEmail   ?string          — LegalSettings data-protection email; $dpoText = it or the «[…]» marker
      $sections   list<{title, paragraphs, items, link}> — the full policy (lang/fa/privacy.php)
    The full policy is a native <details id="policy"> under the design's summary card: «خواندن متن کامل» opens it
    without JS, the text is in the HTML for search engines, and /privacy#policy opens it in current browsers.
    Copy: lang/fa/privacy.php.
--}}
@extends('layouts.app')

@php
    $toolIcons = ['lock', 'eye-off', 'download', 'trash'];
@endphp

@section('hero')
    <section aria-labelledby="privacy-title" class="flex max-w-305 flex-col gap-5.5 px-30 max-lg:max-w-255 pt-14 pb-27.5 max-lg:px-5 max-lg:pt-[30.8px] max-lg:pb-[60.5px]">
        <x-ui.eyebrow tone="dark">{{ __('privacy.hero.eyebrow') }}</x-ui.eyebrow>
        <h1 id="privacy-title" class="m-0 font-display text-[64px] leading-display font-normal text-on-night max-lg:text-[46px] max-sm:text-[34px]">{{ __('privacy.hero.title_before') }}<span class="text-lilac">{{ __('privacy.hero.title_highlight') }}</span>{{ __('privacy.hero.title_after') }}</h1>
        <p class="m-0 max-w-180 text-[19px] leading-loose font-medium text-on-night-muted">{{ __('privacy.hero.lead') }}</p>
    </section>
@endsection

@section('content')
    <x-ui.section aria-label="{{ __('privacy.principles.label') }}" class="flex flex-col gap-10">
        <div class="flex gap-5 max-lg:flex-wrap">
            @foreach (['do' => ['check', 'bg-success-soft', 'text-stage-teen'], 'dont' => ['x', 'bg-danger-soft', 'text-stage-cycle']] as $kind => [$icon, $tileBg, $iconColor])
                <div class="flex flex-1 flex-col gap-3 rounded-5xl border border-line bg-surface p-7 max-lg:basis-75 max-sm:basis-full">
                    <h2 class="m-0 font-sans text-3xl leading-normal font-bold">{{ __("privacy.principles.{$kind}.title") }}</h2>
                    <ul class="m-0 flex list-none flex-col gap-3 p-0">
                        @foreach (__("privacy.principles.{$kind}.items") as $item)
                            <li class="flex items-center gap-2.5 text-[15.5px] font-semibold max-lg:flex-wrap">
                                <span aria-hidden="true" class="flex size-7 shrink-0 items-center justify-center rounded-full {{ $tileBg }}"><x-icon :name="$icon" class="size-[15px] {{ $iconColor }}"/></span>{{ $item }}
                            </li>
                        @endforeach
                    </ul>
                </div>
            @endforeach
        </div>
    </x-ui.section>

    <x-ui.section bg="surface" aria-labelledby="tools-title" class="flex flex-col gap-8">
        <x-ui.section-header :eyebrow="__('privacy.tools.eyebrow')" :title="__('privacy.tools.title')" id="tools-title"/>
        <div class="grid grid-cols-4 gap-4 max-lg:grid-cols-2 max-sm:grid-cols-1">
            @foreach (__('privacy.tools.items') as $i => $tool)
                <div class="flex flex-col gap-2.5 rounded-4xl border border-line bg-surface p-6">
                    <x-icon :name="$toolIcons[$i % 4]" class="size-7.5 text-primary"/>
                    <h3 class="m-0 text-2xl font-bold">{{ $tool['title'] }}</h3>
                    <span class="text-[14.5px] leading-relaxed font-medium text-muted">{{ $tool['text'] }}</span>
                </div>
            @endforeach
        </div>
    </x-ui.section>

    <x-ui.section aria-labelledby="consents-title" class="flex flex-col gap-7">
        <x-ui.section-header :eyebrow="__('privacy.consents.eyebrow')" :title="__('privacy.consents.title')" :lead="__('privacy.consents.lead')" id="consents-title"/>
        <div class="rounded-5xl border border-line bg-surface px-7 py-2">
            @foreach (__('privacy.consents.items') as $i => $consent)
                <x-ui.toggle-row as="h3" class="max-lg:gap-y-0" :title="$consent['title']" :text="$consent['text']" :divided="$i > 0"/>
            @endforeach
        </div>
    </x-ui.section>

    <x-ui.section pad="none" aria-labelledby="policy-title" class="flex flex-col gap-10 pb-24 max-lg:pb-[52.8px]">
        <details id="policy" class="group scroll-mt-6">
            <summary class="grid cursor-pointer list-none grid-cols-[1fr_auto] items-center gap-x-4 rounded-5xl bg-lavender px-8 py-7 max-lg:grid-cols-1 max-lg:gap-y-4 [&::-webkit-details-marker]:hidden">
                <h2 id="policy-title" class="col-start-1 m-0 font-sans text-[19px] leading-normal font-bold">{{ __('privacy.policy.title') }}</h2>
                <span class="col-start-1 block text-md font-semibold text-muted">
                    @if ($updatedAt !== null)
                        {{ __('privacy.policy.updated', ['date' => '']) }}<time datetime="{{ $updatedAt->format('Y-m-d') }}">{{ jdate($updatedAt, 'j F Y') }}</time> ·
                    @endif
                    {{ __('privacy.policy.dpo', ['email' => $dpoText]) }}
                </span>
                <span aria-hidden="true" class="col-start-2 row-span-2 row-start-1 inline-flex h-13.5 items-center gap-2 justify-self-start rounded-full border-[1.5px] border-line px-6.5 text-[15.5px] font-extrabold text-ink group-open:border-primary group-open:text-primary max-lg:col-start-1 max-lg:row-span-1 max-lg:row-start-3">
                    <span class="group-open:hidden">{{ __('privacy.policy.open') }}</span>
                    <span class="hidden group-open:inline">{{ __('privacy.policy.close') }}</span>
                </span>
                <span class="sr-only">{{ __('privacy.policy.open') }}</span>
            </summary>
            <div class="mt-6 flex flex-col gap-8 rounded-5xl border border-line bg-surface p-10 max-sm:p-6">
                @foreach ($sections as $n => $section)
                    <section aria-labelledby="policy-{{ $n + 1 }}" class="flex flex-col gap-3">
                        <h3 id="policy-{{ $n + 1 }}" class="m-0 text-xl font-bold text-ink">{{ fa_digits($n + 1) }}. {{ $section['title'] }}</h3>
                        @foreach ($section['paragraphs'] as $paragraph)
                            <p class="m-0 text-md leading-loose font-medium text-muted">{{ $paragraph }}</p>
                        @endforeach
                        @if ($section['items'] !== [])
                            <ul class="m-0 flex list-disc flex-col gap-2 ps-6 text-md leading-loose font-medium text-muted marker:text-primary">
                                @foreach ($section['items'] as $item)
                                    <li>{{ $item }}</li>
                                @endforeach
                            </ul>
                        @endif
                        @if ($section['link'] !== null)
                            <a href="{{ $section['link']['url'] }}" class="self-start text-md font-extrabold">{{ $section['link']['text'] }}</a>
                        @endif
                    </section>
                @endforeach
                @if ($dpoEmail !== null)
                    <p class="m-0 text-md font-semibold text-ink">{{ __('privacy.policy.dpo', ['email' => '']) }}<a href="mailto:{{ $dpoEmail }}" class="font-extrabold" dir="ltr">{{ $dpoEmail }}</a></p>
                @endif
            </div>
        </details>
    </x-ui.section>
@endsection
