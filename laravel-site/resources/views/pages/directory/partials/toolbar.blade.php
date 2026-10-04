{{--
    Result toolbar: «فیلترها» panel (native <details>, GET form with every amenity), one-tap toggle links (open now,
    the first amenities, cheapest first), the result count and the sort menu (links). Data: $filters, $sorts, $summary.
--}}
@php
    $chip = 'inline-flex h-9.5 shrink-0 cursor-pointer items-center gap-1.5 whitespace-nowrap rounded-full border-[1.5px] px-3.5 text-[12.5px] font-bold transition-colors hover:border-primary';
@endphp
<div class="flex items-center gap-2 max-lg:flex-wrap">
    <details class="group relative">
        <summary @class([$chip, 'list-none [&::-webkit-details-marker]:hidden', $filters['count'] > 0 ? 'border-primary bg-primary/13 text-ink' : 'border-line bg-surface text-muted'])>
            <x-icon name="filter" class="size-[15px]"/>{{ __('directory.filters.label') }}@if ($filters['count'] > 0)<span class="sr-only"> ({{ fa_digits($filters['count']) }})</span>@endif
        </summary>
        <form method="get" action="{{ $filters['action'] }}" class="absolute start-0 top-full z-20 mt-2 flex w-72 flex-col gap-3.5 rounded-3xl border border-line bg-surface p-5 shadow-card">
            <fieldset class="m-0 flex flex-col gap-2.5 border-0 p-0">
                <legend class="mb-2 text-base font-extrabold text-ink">{{ __('directory.filters.panel') }}</legend>
                @foreach ($filters['amenities'] as $amenity)
                    <x-ui.form.checkbox name="amenity[]" :value="$amenity['slug']" :checked="$amenity['checked']">{{ $amenity['label'] }}</x-ui.form.checkbox>
                @endforeach
            </fieldset>
            @foreach ($filters['hidden'] as $name => $value)
                @foreach ((array) $value as $item)
                    <input type="hidden" name="{{ is_array($value) ? $name.'[]' : $name }}" value="{{ $item }}">
                @endforeach
            @endforeach
            <x-ui.button type="submit" size="md">{{ __('directory.filters.apply') }}</x-ui.button>
        </form>
    </details>
    <ul aria-label="{{ __('directory.filters.quick') }}" class="m-0 flex list-none flex-wrap items-center gap-2 p-0">
        @foreach ($filters['toggles'] as $toggle)
            <li><a href="{{ $toggle['href'] }}" @if ($toggle['active']) aria-current="true" @endif @class([$chip, $toggle['active'] ? 'border-primary bg-primary/13 text-ink' : 'border-line bg-surface text-muted'])>{{ $toggle['label'] }}</a></li>
        @endforeach
    </ul>
</div>

<div class="flex items-center justify-between gap-3 max-lg:flex-wrap">
    <p class="m-0 text-md font-bold text-muted" role="status"><b class="text-ink">{{ $summary['count'] }}</b>@if ($summary['where']) {{ $summary['where'] }}@endif</p>
    <details class="relative">
        <summary class="flex cursor-pointer list-none items-center gap-1.5 text-base font-bold text-muted hover:text-ink [&::-webkit-details-marker]:hidden">
            {{ __('directory.sort.label') }} <b class="text-ink">{{ $sorts['label'] }}</b>
            <x-icon name="chevron-left" class="size-3.5 text-muted"/>
        </summary>
        <ul aria-label="{{ __('directory.sort.menu') }}" class="absolute end-0 top-full z-20 m-0 mt-2 flex w-44 list-none flex-col gap-1 rounded-3xl border border-line bg-surface p-2 shadow-card">
            @foreach ($sorts['items'] as $sort)
                <li><a href="{{ $sort['href'] }}" @if ($sort['active']) aria-current="true" @endif @class([
                    'flex h-10 items-center rounded-2xl px-3.5 text-base font-bold text-ink hover:bg-lavender hover:text-ink',
                    'bg-lavender' => $sort['active'],
                ])>{{ $sort['label'] }}</a></li>
            @endforeach
        </ul>
    </details>
</div>
