{{--
    Replaces the design's map column (560×1180, AUDIT §8 — no embedded maps, no map JS): the local `map-directory`
    illustration as a decorative backdrop with crawlable links on top — city landings, the current city's districts and
    the city × category landings that have places. Below 700px it follows the results at full width. Data: $areas.
--}}
@php
    $link = 'flex h-9.5 items-center rounded-full border-[1.5px] px-3.5 text-sm-plus font-bold transition-colors hover:border-primary hover:text-ink';
    $groups = array_values(array_filter([
        ['id' => 'areas-cities', 'title' => __('directory.areas.cities'), 'items' => $areas['cities']],
        ['id' => 'areas-districts', 'title' => $areas['cityName'] ? __('directory.areas.districts', ['city' => $areas['cityName']]) : null, 'items' => $areas['districts']],
        ['id' => 'areas-landings', 'title' => __('directory.areas.landings'), 'items' => $areas['landings']],
    ], static fn (array $group): bool => $group['items'] !== [] && $group['title'] !== null));
@endphp
<aside aria-labelledby="areas-title" class="relative isolate flex h-295 w-140 shrink-0 flex-col justify-between gap-5 overflow-hidden rounded-6xl border border-line p-5 max-sm:h-auto max-sm:min-h-150 max-sm:w-full max-sm:max-w-full">
    <x-illustration name="map-directory" class="absolute inset-0 -z-10 size-full"/>
    <div class="flex flex-col gap-4 rounded-4xl border border-line bg-surface/92 p-5">
        <h2 id="areas-title" class="m-0 text-lg font-extrabold text-ink">{{ __('directory.areas.title') }}</h2>
        @foreach ($groups as $group)
            <nav aria-labelledby="{{ $group['id'] }}" class="flex flex-col gap-2.5">
                <h3 id="{{ $group['id'] }}" class="m-0 text-sm-plus font-extrabold text-muted">{{ $group['title'] }}</h3>
                <ul class="m-0 flex list-none flex-wrap gap-2 p-0">
                    @foreach ($group['items'] as $item)
                        <li><a href="{{ $item['href'] }}" @if ($item['active']) aria-current="page" @endif @class([$link, $item['active'] ? 'border-primary bg-lavender text-ink' : 'border-line bg-surface text-ink'])>{{ $item['label'] }}</a></li>
                    @endforeach
                </ul>
            </nav>
        @endforeach
    </div>
    <p class="m-0 flex items-start gap-2.5 rounded-3xl border border-line bg-surface/92 px-4.5 py-3.5 text-sm-plus leading-relaxed font-semibold text-muted">
        <x-icon name="map-pin" class="mt-0.5 size-4.5 shrink-0 text-primary"/>{{ __('directory.areas.note') }}
    </p>
</aside>
