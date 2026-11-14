{{--
    <x-site.download-band title="ریتمی را رایگان نصب کن" lead="…" :links="$appLinks"/>
    The redesign's `#download` block: ink card (radius 32, p 72/64) inside the page gutters, centred Lalezar 50
    title, lead 18, and four white store buttons (rose icon tile, «دریافت از» + store name). A store whose link is
    not set is skipped; `links` = AppLinksSettings from the controller. Pass `id=""` to drop the anchor.
--}}
@props(['title', 'lead' => null, 'links' => null, 'id' => 'download'])
@php
    $urls = $links instanceof \App\Domain\Settings\Data\AppLinksSettings ? $links->toArray() : (array) ($links ?? []);
    $stores = [];
    foreach (['bazaar' => 'کافه‌بازار', 'myket' => 'مایکت', 'google_play' => 'گوگل‌پلی', 'app_store' => 'نسخه iOS'] as $store => $label) {
        $url = $urls[$store] ?? null;
        if (is_string($url) && preg_match('~^(https?://|/|#)~i', $url) === 1) {
            $stores[] = ['label' => $label, 'url' => $url, 'external' => preg_match('~^https?://~i', $url) === 1];
        }
    }
    $headingId = ($id !== '' ? $id : 'download-band').'-title';
@endphp
<section @if ($id !== '') id="{{ $id }}" @endif aria-labelledby="{{ $headingId }}" {{ $attributes->class('bg-surface px-30 pb-24 max-xl:px-6 max-lg:px-5 max-lg:pb-14') }}>
    <div class="flex flex-col items-center gap-4 rounded-6xl bg-ink px-16 py-18 text-center max-lg:px-8 max-lg:py-12 max-sm:rounded-4xl max-sm:px-5">
        <h2 id="{{ $headingId }}" class="m-0 font-display text-[50px] leading-[1.25] font-normal text-white max-lg:text-d-lg">{{ $title }}</h2>
        @if ($lead)<p class="m-0 text-2xl leading-relaxed font-medium text-on-ink-soft max-sm:text-lg">{{ $lead }}</p>@endif
        @if ($stores !== [])
            <ul class="m-0 mt-5 grid list-none grid-cols-[repeat(4,auto)] justify-center gap-3.5 p-0 max-lg:grid-cols-2 max-sm:w-full max-sm:grid-cols-1" aria-label="دریافت اپ ریتمی">
                @foreach ($stores as $store)
                    <li><a href="{{ $store['url'] }}" @if ($store['external']) rel="noopener" @endif class="flex h-17 min-w-50 items-center justify-center gap-2.5 rounded-xl bg-surface px-5.5 text-ink shadow-store hover:bg-canvas hover:text-ink">
                        <span class="flex size-10 shrink-0 items-center justify-center rounded-md bg-rose"><x-icon name="download" class="size-4.5 text-white"/></span>
                        <span class="flex flex-col text-start leading-[1.35]"><span class="text-[11.5px] font-bold text-muted">دریافت از</span><b class="text-xl">{{ $store['label'] }}</b></span>
                    </a></li>
                @endforeach
            </ul>
        @endif
        {{ $slot }}
    </div>
</section>
