{{--
    The static middle of the home page: stage cards → «یک اپ، پنج بخش» (#how) → two feature splits → help → privacy →
    tools. Copy + routes only (no settings, no per-request data), so HomeController renders it once and caches the
    HTML in the `pages` namespace. Never put per-request or `priority` images in here.
--}}
@php
    $stageCards = [
        ['key' => 'cycle', 'route' => 'stage.cycle', 'icon' => 'drop'],
        ['key' => 'ttc', 'route' => 'stage.ttc', 'icon' => 'target'],
        ['key' => 'pregnancy', 'route' => 'stage.pregnancy', 'icon' => 'egg'],
        ['key' => 'postpartum', 'route' => 'stage.postpartum', 'icon' => 'person'],
        ['key' => 'menopause', 'route' => 'stage.menopause', 'icon' => 'moon'],
        ['key' => 'teen', 'route' => 'stage.teen', 'icon' => 'sprout'],
    ];
    $howTiles = ['today' => 'home', 'stage' => 'calendar', 'voice' => 'mic', 'services' => 'grid', 'me' => 'user'];
    $help = [];
    foreach ([
        'doctor' => [route('services'), 'stethoscope', 'postpartum'],
        'directory' => [route('directory.index'), 'map', 'primary'],
        'shop' => [route('shop.index'), 'store', 'muted'],
    ] as $helpKey => [$helpHref, $helpIcon, $helpColor]) {
        $help[] = ['href' => $helpHref, 'icon' => $helpIcon, 'color' => $helpColor] + __("home.help.items.{$helpKey}");
    }
    $privacyItems = ['lock' => 'lock', 'hidden' => 'eye-off', 'export' => 'download', 'delete' => 'trash'];
    $tools = [
        ['key' => 'due_date', 'anchor' => 'due-date', 'icon' => 'calculator', 'color' => 'pregnancy', 'where' => 'on_site'],
        ['key' => 'fertility', 'anchor' => 'fertility', 'icon' => 'target', 'color' => 'ttc', 'where' => 'on_site'],
        ['key' => 'hospital_bag', 'anchor' => 'hospital-bag', 'icon' => 'bag', 'color' => 'pregnancy', 'where' => 'in_app'],
        ['key' => 'sisemoni', 'anchor' => 'sisemoni', 'icon' => 'check-square', 'color' => 'primary', 'where' => 'in_app'],
    ];
@endphp

<x-ui.section id="stages" aria-labelledby="home-stages-title" class="flex flex-col gap-10">
    <x-ui.section-header align="center" id="home-stages-title" :eyebrow="__('home.stages.eyebrow')" :title="__('home.stages.title')" :lead="__('home.stages.lead')"/>
    <div class="grid grid-cols-3 gap-4.5 max-sm:grid-cols-1">
        @foreach ($stageCards as $card)
            <x-cards.stage :href="route($card['route'])" :icon="$card['icon']" :color="$card['key']" :title="__('home.stages.items.'.$card['key'].'.title')" :text="__('home.stages.items.'.$card['key'].'.text')" :cta="__('home.stages.cta')"/>
        @endforeach
    </div>
</x-ui.section>

{{-- Design fix: the card kept its 120px side margins on phones (content ~22px wide); ≤700 it gets 20px like #download. --}}
<section id="how" aria-labelledby="home-how-title" class="mx-30 flex flex-col gap-8 rounded-7xl bg-night bg-hero-glow p-16 max-sm:mx-5 max-sm:rounded-4xl max-sm:p-6">
    <h2 id="home-how-title" class="m-0 font-display text-d-lg leading-heading font-normal text-on-night">{{ __('home.how.title') }}</h2>
    <p class="m-0 text-xl leading-loose font-medium text-on-night-muted">{{ __('home.how.lead') }}</p>
    <ul class="m-0 flex list-none gap-3.5 p-0 max-lg:flex-wrap">
        @foreach ($howTiles as $tileKey => $tileIcon)
            <li class="flex flex-1 flex-col gap-2.5 rounded-4xl border border-night-line bg-surface/6 p-5.5 max-lg:basis-75 max-sm:basis-full">
                <x-ui.icon-tile :icon="$tileIcon" color="lilac" size="sm" shape="circle"/>
                <h3 class="m-0 text-2xl font-bold text-on-night">{{ __("home.how.items.{$tileKey}.title") }}</h3>
                <span class="text-base leading-relaxed font-medium text-on-night-muted">{{ __("home.how.items.{$tileKey}.text") }}</span>
            </li>
        @endforeach
    </ul>
</section>

@include('pages.home.split', ['key' => 'uncertainty', 'screen' => 'cycle-today', 'mediaFirst' => false, 'bg' => 'none'])
@include('pages.home.split', ['key' => 'family', 'screen' => 'companion', 'mediaFirst' => true, 'bg' => 'surface'])

<x-stage.help-block :items="$help" :eyebrow="__('home.help.eyebrow')" :title="__('home.help.title')" id="home-help" bg="none"/>

<x-ui.section bg="surface" aria-labelledby="home-privacy-title" class="flex items-center gap-12 max-lg:flex-wrap">
    <div class="flex flex-1 flex-col gap-4 max-lg:basis-75 max-sm:basis-full">
        <x-ui.eyebrow>{{ __('home.privacy.eyebrow') }}</x-ui.eyebrow>
        <h2 id="home-privacy-title" class="m-0 font-display text-d-lg leading-heading font-normal text-ink">{{ __('home.privacy.title') }}</h2>
        <p class="m-0 text-xl leading-loose font-medium text-muted">{{ __('home.privacy.text') }}</p>
        <a href="{{ route('privacy') }}" class="flex items-center gap-1.5 text-md font-extrabold">{{ __('home.privacy.link') }}<x-icon name="arrow-left" class="size-4 text-primary"/></a>
    </div>
    <div class="grid flex-1 grid-cols-2 gap-3.5 max-lg:basis-75 max-sm:basis-full max-sm:grid-cols-1">
        @foreach ($privacyItems as $privacyKey => $privacyIcon)
            <x-cards.value variant="compact" :icon="$privacyIcon" :title="__('home.privacy.items.'.$privacyKey)"/>
        @endforeach
    </div>
</x-ui.section>

<x-ui.section pad="none" aria-labelledby="home-tools-title" class="flex flex-col gap-8 pt-10 pb-24 max-lg:pt-6 max-lg:pb-[52.8px]">
    <x-ui.section-header id="home-tools-title" :eyebrow="__('home.tools.eyebrow')" :title="__('home.tools.title')"/>
    <div class="grid grid-cols-4 gap-4 max-lg:grid-cols-2 max-sm:grid-cols-1">
        @foreach ($tools as $tool)
            <x-cards.feature :href="route('tools').'#'.$tool['anchor']" :icon="$tool['icon']" :color="$tool['color']" :title="__('home.tools.items.'.$tool['key'].'.title')" :text="__('home.tools.items.'.$tool['key'].'.text')" :where="__('home.tools.'.$tool['where'])"/>
        @endforeach
    </div>
</x-ui.section>
