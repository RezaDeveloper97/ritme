{{--
    <x-directory.address :line="'[آدرس] · ونک، تهران'" pin="آب‌پری" :links="[['label' => 'نشان', 'href' => 'https://neshan.org/…', 'external' => true]]"/>
    «آدرس» block (L5-03). No embedded map (tasks/README.md Maps decision): a local static illustration (map-place) with
    the place's name pinned on it, the address line and "open in maps" deep links (geo: URI for the phone's map app,
    Neshan, Balad, Google Maps). The links are plain anchors — nothing is loaded from those hosts.
--}}
@props(['line' => null, 'links' => [], 'pin'])
<figure class="relative m-0 box-content h-70 overflow-hidden rounded-5xl border border-line">
    <x-illustration name="map-place" width="100%" height="100%" class="block size-full"/>
    <div aria-hidden="true" class="absolute end-85 top-22.5 flex flex-col items-center max-sm:hidden">
        <span class="flex h-10 items-center gap-1.5 rounded-full border-2 border-primary bg-primary ps-3 pe-1.5 text-xs font-extrabold whitespace-nowrap text-white shadow-chip">
            <span class="flex size-7 items-center justify-center rounded-full bg-surface/13"><x-icon name="map-pin" class="size-[15px] text-white"/></span>{{ $pin }}
        </span>
        <span class="-mt-1.5 size-2.5 rotate-45 rounded-[2px] bg-primary"></span>
    </div>
    <figcaption class="sr-only">{{ __('directory.place.address.map', ['name' => $pin]) }}</figcaption>
</figure>
<div class="flex items-center justify-between gap-4 max-lg:flex-wrap">
    <address class="flex gap-2 text-md font-semibold text-muted not-italic">
        <x-icon name="map-pin" class="mt-1 size-4.5 shrink-0 text-muted"/>{{ $line ?? __('directory.place.address.missing') }}
    </address>
    @if ($links !== [])
        <nav aria-label="{{ __('directory.place.address.open_in') }}" class="flex flex-wrap items-center gap-2">
            @foreach (array_slice($links, 1) as $link)
                <a href="{{ $link['href'] }}" @if ($link['external']) rel="noopener nofollow" target="_blank" @endif
                   class="inline-flex h-10 items-center rounded-full border-[1.5px] border-line px-4 text-sm-plus font-extrabold text-ink hover:border-primary">{{ $link['label'] }}</a>
            @endforeach
            <a href="{{ $links[0]['href'] }}" class="box-content inline-flex h-13.5 items-center gap-2 rounded-full border-[1.5px] border-line bg-transparent px-6.5 text-[15.5px] font-extrabold text-ink hover:border-primary">
                <x-icon name="navigate" class="size-4.5 text-ink"/>{{ __('directory.place.actions.directions') }}<span class="sr-only"> — {{ $links[0]['label'] }}</span>
            </a>
        </nav>
    @endif
</div>
