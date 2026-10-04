{{--
    Social responsibility page (L3-08), design/html/social-responsibility.html. Data from
    App\Http\Controllers\SocialResponsibilityController:
      $freeItems        list<string>              — «همیشه رایگان» rows
      $programs         list<{title, text}>       — «فراتر از اپ» (placeholders until real programs exist)
      $transparencyUrl  ?string                   — «مشاهده گزارش» only when set
      $appLinks         AppLinksSettings, $qrUrl  — the #download card
    Copy: lang/fa/social.php. Donations (AUDIT §8): no gateway → contact CTA, no amounts.
--}}
@extends('layouts.app')

@php
    $freeStyles = [
        ['icon' => 'drop', 'tint' => 'bg-stage-cycle/10 text-stage-cycle'],
        ['icon' => 'mic', 'tint' => 'bg-primary/10 text-primary'],
        ['icon' => 'egg', 'tint' => 'bg-stage-pregnancy/10 text-stage-pregnancy'],
        ['icon' => 'syringe', 'tint' => 'bg-stage-postpartum/10 text-stage-postpartum'],
        ['icon' => 'moon', 'tint' => 'bg-primary/10 text-primary'],
        ['icon' => 'sprout', 'tint' => 'bg-stage-teen/10 text-stage-teen'],
        ['icon' => 'lock', 'tint' => 'bg-primary/10 text-primary'],
    ];
    $programStyles = [
        ['icon' => 'sprout', 'color' => 'teen'],
        ['icon' => 'stethoscope', 'color' => 'cycle'],
        ['icon' => 'download', 'color' => 'primary'],
        ['icon' => 'book', 'color' => 'pregnancy'],
    ];
    $outlineButton = 'inline-flex h-13.5 items-center gap-2 self-start rounded-full border-[1.5px] border-line px-6.5 text-[15.5px] font-extrabold text-ink transition-colors hover:border-primary hover:text-primary';
@endphp

@section('hero')
    <section aria-labelledby="social-title" class="flex flex-col items-center gap-6 px-30 pt-14 pb-27.5 text-center max-lg:px-5 max-lg:pt-[30.8px] max-lg:pb-[60.5px]">
        {{-- Design pill colour #FF9BB3 is outside the palette: phase-period lightened with white (same hue). --}}
        <span class="inline-flex items-center gap-2 rounded-3xl border border-phase-period/40 bg-phase-period/14 px-4 py-2 text-base font-extrabold text-[color-mix(in_srgb,var(--color-phase-period)_67%,var(--color-white))]"><x-icon name="heart" class="size-4"/>{{ __('social.hero.eyebrow') }}</span>
        <h1 id="social-title" class="m-0 font-display text-d-4xl leading-display font-normal text-on-night">{{ __('social.hero.title_before') }}<span class="text-lilac">{{ __('social.hero.title_highlight') }}</span>{{ __('social.hero.title_after') }}</h1>
        <p class="m-0 max-w-190 text-center text-[19px] leading-loose font-medium text-on-night-muted max-sm:max-w-full">{{ __('social.hero.lead') }}</p>
    </section>
@endsection

@section('content')
    <x-ui.section aria-labelledby="free-title" class="flex flex-col gap-10">
        <div class="flex items-start gap-14 max-lg:flex-wrap">
            <div class="flex flex-1 flex-col gap-4 max-lg:basis-75 max-sm:basis-full">
                <x-ui.section-header :eyebrow="__('social.free.eyebrow')" :title="__('social.free.title')" :lead="__('social.free.lead')" id="free-title" class="gap-4"/>
                <a href="{{ route('plus') }}" class="self-start text-md font-extrabold">{{ __('social.free.link') }}</a>
            </div>
            <ul aria-label="{{ __('social.free.list_label') }}" class="m-0 grid flex-[1.2_1_0] list-none gap-2.5 p-0">
                @foreach ($freeItems as $i => $item)
                    @php $style = $freeStyles[$i % count($freeStyles)]; @endphp
                    <li class="flex items-center gap-3.5 rounded-[22px] border border-line bg-surface px-5 py-4.5 max-lg:flex-wrap">
                        <span aria-hidden="true" class="flex size-11 shrink-0 items-center justify-center rounded-full {{ $style['tint'] }}"><x-icon :name="$style['icon']" class="size-5.5"/></span>
                        <b class="grow text-lg">{{ $item }}</b>
                        <span class="rounded-lg bg-info-soft px-3 py-1.5 text-sm font-extrabold text-stage-postpartum">{{ __('social.free.badge') }}</span>
                    </li>
                @endforeach
            </ul>
        </div>
    </x-ui.section>

    <x-ui.section bg="surface" aria-labelledby="programs-title" class="flex flex-col items-center gap-10">
        <x-ui.section-header :eyebrow="__('social.programs.eyebrow')" :title="__('social.programs.title')" :lead="__('social.programs.lead')" align="center" id="programs-title"/>
        <div class="grid grid-cols-4 gap-4.5 max-lg:grid-cols-2 max-sm:grid-cols-1">
            @foreach ($programs as $i => $program)
                @php $style = $programStyles[$i % count($programStyles)]; @endphp
                <x-cards.value :icon="$style['icon']" :color="$style['color']" :title="$program['title']" :text="$program['text']"/>
            @endforeach
        </div>
    </x-ui.section>

    <x-ui.section as="div" class="flex flex-col gap-10">
        <div class="flex gap-6 max-lg:flex-wrap">
            <section aria-labelledby="support-title" class="flex flex-1 flex-col gap-3.5 rounded-6xl bg-lavender p-9 max-lg:basis-75 max-sm:basis-full">
                <x-icon name="heart" class="size-9 text-stage-cycle"/>
                <h2 id="support-title" class="m-0 font-display text-[30px] leading-heading font-normal text-ink">{{ __('social.support.title') }}</h2>
                <p class="m-0 text-lg leading-loose font-medium text-muted">{{ __('social.support.text') }}</p>
                <a href="{{ route('contact') }}" class="{{ $outlineButton }} bg-surface"><x-icon name="arrow-left" class="size-4.5"/>{{ __('social.support.cta') }}</a>
            </section>
            <section aria-labelledby="partner-title" class="flex flex-1 flex-col gap-3.5 rounded-6xl border border-line bg-surface p-9 max-lg:basis-75 max-sm:basis-full">
                <x-icon name="users" class="size-9 text-stage-postpartum"/>
                <h2 id="partner-title" class="m-0 font-display text-[30px] leading-heading font-normal text-ink">{{ __('social.partner.title') }}</h2>
                <p class="m-0 text-lg leading-loose font-medium text-muted">{{ __('social.partner.text') }}</p>
                <a href="{{ route('contact') }}" class="{{ $outlineButton }}"><x-icon name="arrow-left" class="size-4.5"/>{{ __('social.partner.cta') }}</a>
            </section>
        </div>
    </x-ui.section>

    <x-ui.section bg="surface" aria-label="{{ __('social.quote.label') }}" class="flex flex-col items-center gap-5">
        <figure class="m-0 mx-auto flex max-w-220 flex-col items-center gap-4.5 text-center">
            <x-icon name="heart" class="size-10 text-stage-cycle"/>
            <blockquote class="m-0 font-display text-d-md leading-[1.7]"><p class="m-0">«{{ __('social.quote.text') }}»</p></blockquote>
            <figcaption class="text-md font-bold text-muted">{{ __('social.quote.name') }} · {{ __('social.quote.role') }}</figcaption>
        </figure>
    </x-ui.section>

    <x-ui.section pad="none" aria-labelledby="transparency-title" class="flex flex-col gap-10 pt-10 pb-24 max-lg:pt-6 max-lg:pb-[52.8px]">
        <div class="flex items-center justify-between gap-4 rounded-5xl border border-line bg-surface px-8 py-7 max-lg:flex-wrap">
            <div>
                <h2 id="transparency-title" class="m-0 font-sans text-[19px] leading-normal font-bold">{{ __('social.transparency.title') }}</h2>
                <p class="m-0 text-md font-semibold text-muted">{{ __('social.transparency.text') }}</p>
            </div>
            @if ($transparencyUrl !== null)
                <a href="{{ $transparencyUrl }}" class="{{ $outlineButton }}">{{ __('social.transparency.cta') }}</a>
            @endif
        </div>
    </x-ui.section>

    <x-ui.app-cta :title="__('social.app_cta.title')" :lead="__('social.app_cta.lead')" :links="$appLinks" :qr-url="$qrUrl"/>
@endsection
