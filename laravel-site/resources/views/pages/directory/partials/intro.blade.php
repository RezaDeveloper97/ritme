{{--
    Header band: eyebrow, the page's h1, lead, the GET search form (text · «کجا؟» city/district · child age) and the
    category chips (links; on a city they point at the indexable city × category landings). Data: $intro,
    $breadcrumbs, $search, $chips from ListPlacesController.
--}}
<section class="flex flex-col gap-6 bg-linear-180/srgb from-white to-canvas px-30 pt-12 pb-8 max-lg:px-5 max-lg:pt-[26.4px]">
    @if ($breadcrumbs)
        <x-ui.breadcrumbs :items="$breadcrumbs"/>
    @endif
    <span class="text-base font-extrabold text-primary">{{ $intro['eyebrow'] }}</span>
    <h1 class="m-0 font-display text-[52px] leading-display font-normal text-ink max-lg:text-[38px] max-sm:text-[30px]">{{ $intro['title'] }}</h1>
    <p class="m-0 max-w-190 text-xl leading-loose font-medium text-muted max-sm:w-full max-sm:max-w-full">{{ $intro['lead'] }}</p>

    <form role="search" method="get" action="{{ $search['action'] }}" aria-label="{{ __('directory.search.label') }}"
          class="flex items-center rounded-7xl border border-line bg-surface p-2 shadow-search max-lg:flex-wrap max-sm:rounded-4xl">
        <label class="flex min-w-0 flex-[3_1_0] flex-col gap-0.5 rounded-full border-e border-e-line px-5.5 py-2.5 focus-within:ring-2 focus-within:ring-primary max-sm:basis-full max-sm:border-e-0">
            <span class="text-xs font-extrabold text-ink">{{ __('directory.search.what') }}</span>
            <span class="flex items-center gap-2 text-md font-semibold text-muted">
                <x-icon name="search" class="size-4.5 shrink-0 text-muted"/>
                <input type="search" name="q" value="{{ $search['q'] }}" maxlength="80" autocomplete="off" enterkeyhint="search"
                       placeholder="{{ __('directory.search.what_placeholder') }}"
                       class="w-full min-w-0 border-0 bg-transparent p-0 text-md font-semibold text-ink outline-none placeholder:text-muted">
            </span>
        </label>
        <label class="flex min-w-0 flex-[2_1_0] flex-col gap-0.5 rounded-full border-e border-e-line px-5.5 py-2.5 focus-within:ring-2 focus-within:ring-primary max-sm:basis-full max-sm:border-e-0">
            <span class="text-xs font-extrabold text-ink">{{ __('directory.search.where') }}</span>
            <span class="flex items-center gap-2 text-md font-semibold text-muted">
                <x-icon name="map-pin" class="size-4.5 shrink-0 text-muted"/>
                <select name="where" class="w-full min-w-0 cursor-pointer appearance-none border-0 bg-transparent p-0 text-md font-semibold text-ink outline-none">
                    <option value="">{{ __('directory.search.where_any') }}</option>
                    @foreach ($search['whereOptions'] as $value => $label)
                        <option value="{{ $value }}" @selected($search['where'] === (string) $value)>{{ $label }}</option>
                    @endforeach
                </select>
            </span>
        </label>
        <label class="flex min-w-0 flex-[2_1_0] flex-col gap-0.5 rounded-full border-e border-e-line px-5.5 py-2.5 focus-within:ring-2 focus-within:ring-primary max-sm:basis-full max-sm:border-e-0">
            <span class="text-xs font-extrabold text-ink">{{ __('directory.search.age') }}</span>
            <span class="flex items-center gap-2 text-md font-semibold text-muted">
                <x-icon name="person" class="size-4.5 shrink-0 text-muted"/>
                <select name="age" class="w-full min-w-0 cursor-pointer appearance-none border-0 bg-transparent p-0 text-md font-semibold text-ink outline-none">
                    <option value="">{{ __('directory.search.age_any') }}</option>
                    @foreach ($search['ageOptions'] as $value => $label)
                        <option value="{{ $value }}" @selected($search['age'] === (string) $value)>{{ $label }}</option>
                    @endforeach
                </select>
            </span>
        </label>
        @foreach ($search['hidden'] as $name => $value)
            @foreach ((array) $value as $item)
                <input type="hidden" name="{{ is_array($value) ? $name.'[]' : $name }}" value="{{ $item }}">
            @endforeach
        @endforeach
        <button type="submit" class="flex h-15 shrink-0 items-center gap-2 rounded-full border-0 bg-primary px-7.5 text-lg font-extrabold text-white hover:bg-primary-hover max-sm:w-full max-sm:justify-center">
            <x-icon name="search" class="size-5 text-white"/>{{ __('directory.search.submit') }}
        </button>
    </form>

    <x-ui.chip-nav :label="__('directory.chips_label')" :items="$chips" size="lg"/>
</section>
