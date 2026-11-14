{{--
    Home hero (design Main.html): rose-soft band with the dot ring, text side (eyebrow, h1, store buttons from settings,
    «امکانات» link) and the photo card with two floating chips. Rendered per request (NOT in the cached fragment): the
    photo is the LCP image (priority). Data: $heroPhoto MediaData|null, $appLinks AppLinksSettings, $heroAlt string.
--}}
@php
    $urls = $appLinks->toArray();
    $stores = [];
    foreach (['bazaar' => ['کافه‌بازار', 'get_from'], 'google_play' => ['گوگل‌پلی', 'get_from'], 'app_store' => ['اپ‌استور', 'download_from']] as $store => [$label, $verb]) {
        $url = $urls[$store] ?? null;
        if (is_string($url) && preg_match('~^(https?://|/|#)~i', $url) === 1) {
            $stores[] = ['label' => $label, 'verb' => __("home.hero.{$verb}"), 'url' => $url, 'external' => preg_match('~^https?://~i', $url) === 1];
        }
    }
@endphp
<section aria-labelledby="home-title" class="px-6 pt-6 max-sm:px-3 max-sm:pt-3">
    <div class="relative overflow-hidden rounded-6xl bg-danger-soft max-sm:rounded-4xl">
        <x-home.ring class="absolute top-1/2 -end-9.75 size-170 -translate-y-1/2 max-sm:top-0 max-sm:-end-50 max-sm:size-100 max-sm:-translate-y-1/2"/>
        <div class="relative flex items-center gap-18 px-24 py-18 max-xl:gap-10 max-xl:px-10 max-lg:flex-col max-lg:items-stretch max-lg:gap-8 max-sm:gap-5 max-sm:px-5 max-sm:pt-8 max-sm:pb-5">
            <div class="flex flex-1 flex-col gap-6 max-sm:gap-5">
                <span class="text-base font-extrabold text-rose">{{ __('home.hero.eyebrow') }}</span>
                <h1 id="home-title" class="m-0 font-display text-[60px] leading-[1.2] font-normal text-ink max-sm:text-[38px] max-sm:leading-tight">
                    {{ __('home.hero.title_line_1') }} <br class="max-sm:hidden">{{ __('home.hero.title_line_2') }}
                </h1>
                @if ($stores !== [])
                    <div class="flex flex-col gap-2.5">
                        <b class="text-md text-ink max-sm:text-base">{{ __('home.hero.download') }}</b>
                        <ul class="m-0 grid list-none grid-cols-[repeat(3,auto)] justify-start gap-2.5 p-0 max-sm:grid-cols-[repeat(3,minmax(0,1fr))] max-sm:gap-2">
                            @foreach ($stores as $store)
                                <li><a href="{{ $store['url'] }}" @if ($store['external']) rel="noopener" @endif class="flex h-14 items-center justify-center gap-2.5 rounded-lg bg-ink px-5 text-white hover:bg-ink/90 hover:text-white max-sm:h-13 max-sm:gap-1.5 max-sm:px-2.5">
                                    <x-icon name="download" class="size-4.5 shrink-0 max-sm:hidden"/>
                                    <span class="flex flex-col leading-[1.3]"><span class="text-[11px] font-semibold opacity-75 max-sm:text-[10px]">{{ $store['verb'] }}</span><b class="text-[15.5px] max-sm:text-[13.5px]">{{ $store['label'] }}</b></span>
                                </a></li>
                            @endforeach
                        </ul>
                    </div>
                @endif
                <div><x-site.text-link href="#features">{{ __('home.hero.features') }}</x-site.text-link></div>
            </div>
            <div class="relative w-82.5 shrink-0 me-10 max-lg:me-0 max-lg:self-center max-sm:w-full">
                <span class="relative block h-110 overflow-hidden rounded-4xl bg-canvas max-sm:h-90">
                    @if ($heroPhoto)
                        <x-picture :media="$heroPhoto" :alt="$heroAlt" priority sizes="(max-width: 700px) 100vw, 330px" class="size-full object-cover"/>
                    @endif
                </span>
                <div class="absolute top-16 -start-7 flex flex-col gap-0.5 rounded-xl bg-surface px-4 py-3 shadow-pop max-sm:-start-1">
                    <b class="text-base text-ink">{{ __('home.hero.chip_period_title') }}</b>
                    <span class="text-sm font-semibold text-muted">{{ __('home.hero.chip_period_text') }}</span>
                </div>
                <div class="absolute bottom-18 -end-7 flex flex-col gap-0.5 rounded-xl bg-surface px-4 py-3 shadow-pop max-sm:-end-1">
                    <b class="text-base text-ink">{{ __('home.hero.chip_voice_title') }}</b>
                    <span class="text-sm font-semibold text-muted">{{ __('home.hero.chip_voice_text') }}</span>
                </div>
            </div>
        </div>
    </div>
</section>
