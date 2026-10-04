{{--
    About page (L3-08), design/html/about.html. Data from App\Http\Controllers\AboutController:
      $stats     list<{value, label}>  — «ریتمی در چند عدد» (placeholders until the owner supplies real numbers)
      $values    list<{title, text}>   — «چطور تصمیم می‌گیریم»
      $redLines  list<string>          — «کارهایی که هرگز نمی‌کنیم»
      $team      list<{name, role, media: int|null}> — photo via <x-picture> once a media id is set
      $careersUrl ?string              — «فرصت‌های شغلی» card is a link only when set (no href="#")
    Copy: lang/fa/about.php. Anchor #review-policy = scientific council / medical review note (E-E-A-T).
--}}
@extends('layouts.app')

@php
    $valueStyles = [
        ['icon' => 'sparkle', 'color' => 'primary'],
        ['icon' => 'heart', 'color' => 'cycle'],
        ['icon' => 'lock', 'color' => 'postpartum'],
    ];
    $cardLink = 'flex flex-1 flex-col gap-2.5 rounded-5xl border border-line bg-surface p-7 text-ink max-lg:basis-75 max-sm:basis-full';
@endphp

@section('hero')
    <section aria-labelledby="about-title" class="flex max-w-305 flex-col gap-5.5 px-30 max-lg:max-w-255 pt-14 pb-27.5 max-lg:px-5 max-lg:pt-[30.8px] max-lg:pb-[60.5px]">
        <x-ui.eyebrow tone="dark">{{ __('about.hero.eyebrow') }}</x-ui.eyebrow>
        <h1 id="about-title" class="m-0 font-display text-[64px] leading-display font-normal text-on-night max-lg:text-[46px] max-sm:text-[34px]"><span class="text-lilac">{{ __('about.hero.title_highlight') }}</span>{{ __('about.hero.title_rest') }}</h1>
        <p class="m-0 max-w-190 text-[19px] leading-loose font-medium text-on-night-muted max-sm:max-w-full">{{ __('about.hero.lead') }}</p>
    </section>
@endsection

@section('content')
    <x-ui.section aria-labelledby="story-title" class="flex flex-col gap-10">
        <div class="flex items-start gap-14 max-lg:flex-wrap">
            <div class="flex flex-1 flex-col gap-4 max-lg:basis-75 max-sm:basis-full">
                <x-ui.section-header :eyebrow="__('about.story.eyebrow')" :title="__('about.story.title')" id="story-title" class="gap-4"/>
                @foreach (__('about.story.paragraphs') as $paragraph)
                    <p class="m-0 text-xl leading-loose font-medium text-muted">{{ $paragraph }}</p>
                @endforeach
            </div>
            <dl aria-label="{{ __('about.story.stats_label') }}" class="m-0 grid flex-1 grid-cols-2 gap-3.5 max-lg:basis-75 max-sm:basis-full max-sm:grid-cols-1">
                @foreach ($stats as $stat)
                    <div class="flex flex-col-reverse justify-end gap-1.5 rounded-4xl border border-line bg-surface p-5.5">
                        <dt class="text-base font-bold text-muted">{{ $stat['label'] }}</dt>
                        <dd class="m-0 font-display text-[36px] text-primary">{{ $stat['value'] }}</dd>
                    </div>
                @endforeach
            </dl>
        </div>
    </x-ui.section>

    <x-ui.section bg="surface" aria-labelledby="values-title" class="flex flex-col items-center gap-10">
        <x-ui.section-header :eyebrow="__('about.values.eyebrow')" :title="__('about.values.title')" align="center" id="values-title"/>
        <div class="grid grid-cols-3 gap-4.5 max-sm:grid-cols-1">
            @foreach ($values as $i => $value)
                <x-cards.value :icon="$valueStyles[$i % 3]['icon']" :color="$valueStyles[$i % 3]['color']" :title="$value['title']" :text="$value['text']"/>
            @endforeach
        </div>
    </x-ui.section>

    <x-ui.section aria-labelledby="red-lines-title" class="flex flex-col gap-10">
        <div class="flex items-start gap-12 max-lg:flex-wrap">
            <x-ui.section-header :eyebrow="__('about.red_lines.eyebrow')" :title="__('about.red_lines.title')" :lead="__('about.red_lines.lead')" id="red-lines-title" class="flex-1 gap-3.5 max-lg:basis-75 max-sm:basis-full"/>
            <ul class="m-0 grid flex-[1.2_1_0] list-none grid-cols-2 gap-3 p-0 max-sm:grid-cols-1">
                @foreach ($redLines as $line)
                    <li class="flex items-center gap-3 rounded-3xl border border-line bg-surface p-4.5 text-[15.5px] font-bold max-lg:flex-wrap">
                        <span aria-hidden="true" class="flex size-9 shrink-0 items-center justify-center rounded-full bg-danger-soft"><x-icon name="x" class="size-4.5 text-stage-cycle"/></span>{{ $line }}
                    </li>
                @endforeach
            </ul>
        </div>
    </x-ui.section>

    <x-ui.section bg="surface" aria-labelledby="team-title" class="flex flex-col items-center gap-10">
        <x-ui.section-header :eyebrow="__('about.team.eyebrow')" :title="__('about.team.title')" align="center" id="team-title"/>
        <ul class="m-0 grid list-none grid-cols-4 gap-6 p-0 max-lg:grid-cols-2 max-sm:grid-cols-1">
            @foreach ($team as $member)
                <li class="flex flex-col items-center gap-2.5 text-center">
                    @if ($member['media'] !== null)
                        <x-picture :media="$member['media']" :alt="$member['name']" sizes="140px" class="size-35 rounded-full object-cover"/>
                    @else
                        <span class="flex size-35 items-center justify-center rounded-full border border-dashed border-primary bg-lavender text-sm text-muted">{{ __('about.team.photo_placeholder') }}</span>
                    @endif
                    <h3 class="m-0 text-xl font-bold">{{ $member['name'] }}</h3>
                    <span class="text-base font-semibold text-muted">{{ $member['role'] }}</span>
                </li>
            @endforeach
        </ul>
        <div id="review-policy" class="flex scroll-mt-6 items-center gap-5 rounded-5xl bg-lavender p-7 max-lg:flex-wrap">
            <x-icon name="stethoscope" class="size-9 shrink-0 text-primary"/>
            <div class="grow">
                <h3 class="m-0 text-[19px] font-bold">{{ __('about.team.council.title') }}</h3>
                <p class="m-0 text-md leading-relaxed font-semibold text-muted">{{ __('about.team.council.text') }}</p>
            </div>
        </div>
    </x-ui.section>

    <x-ui.section as="nav" pad="none" aria-label="{{ __('about.links.label') }}" class="flex flex-col gap-10 pt-10 pb-24 max-lg:pt-6 max-lg:pb-[52.8px]">
        <div class="flex gap-5 max-lg:flex-wrap">
            <a href="{{ route('social-responsibility') }}" class="{{ $cardLink }}">
                <x-icon name="heart" class="size-7.5 text-primary"/>
                <b class="text-[19px]">{{ __('about.links.social.title') }}</b>
                <span class="text-[14.5px] font-semibold text-muted">{{ __('about.links.social.text') }}</span>
            </a>
            @if ($careersUrl !== null)
                <a href="{{ $careersUrl }}" class="{{ $cardLink }}">
            @else
                <div class="{{ $cardLink }}">
            @endif
                <x-icon name="users" class="size-7.5 text-primary"/>
                <b class="text-[19px]">{{ __('about.links.careers.title') }}</b>
                <span class="text-[14.5px] font-semibold text-muted">{{ __('about.links.careers.text') }}</span>
            @if ($careersUrl !== null)
                </a>
            @else
                </div>
            @endif
            <a href="{{ route('contact') }}" class="{{ $cardLink }}">
                <x-icon name="message" class="size-7.5 text-primary"/>
                <b class="text-[19px]">{{ __('about.links.press.title') }}</b>
                <span class="text-[14.5px] font-semibold text-muted">{{ __('about.links.press.text') }}</span>
            </a>
        </div>
    </x-ui.section>

    <x-ui.promise-banner href="{{ route('social-responsibility') }}"/>
@endsection
