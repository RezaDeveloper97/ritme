{{--
    <x-shop.filters :filters="$filters" :reset-url="$resetUrl"/>
    Side panel of the category listing (shop-list.html «فیلترها», 280px). Category links (crawlable, indexable pages)
    + ONE GET form (no JS): sizes and colours as checkbox pills / swatches (CSS `has-checked:`), price range in tomans,
    brands, «فقط کالاهای موجود» as a checkbox styled like the design's switch, and an apply button. The controller
    301s the submitted query to its canonical form. Colour swatches are SVG circles filled with the variant's hex
    (data, validated #RRGGBB — no inline style). $filters: see Shop\CategoryController::filters().
--}}
@props(['filters', 'resetUrl'])
@php
    $block = 'm-0 flex flex-col gap-3 border-0 border-t border-line px-0 py-5';
    $legend = 'float-start mb-3 w-full p-0 text-md font-bold text-ink';
    $pill = 'flex h-10 cursor-pointer items-center rounded-full border-[1.5px] border-line px-3.5 text-[12.5px] font-bold text-muted hover:border-primary has-checked:border-primary has-checked:bg-primary/13 has-checked:text-ink has-focus-visible:outline-2 has-focus-visible:outline-primary';
@endphp
<section aria-labelledby="filters-title" {{ $attributes->class('flex w-70 shrink-0 flex-col max-sm:w-full') }}>
    <div class="flex items-center justify-between pb-4">
        <h2 id="filters-title" class="m-0 text-xl font-bold text-ink">{{ __('shop.filters.title') }}</h2>
        @if ($filters['filtered'])
            <a href="{{ $resetUrl }}" class="text-sm-plus font-extrabold">{{ __('shop.filters.clear') }}</a>
        @endif
    </div>

    @if ($filters['categories'] !== [])
        <nav aria-labelledby="filter-categories" class="{{ $block }}">
            <h3 id="filter-categories" class="m-0 text-md font-bold text-ink">{{ __('shop.filters.categories') }}</h3>
            <ul class="m-0 flex list-none flex-col gap-3 p-0">
                @foreach ($filters['categories'] as $item)
                    <li><a href="{{ $item['href'] }}" @if ($item['active']) aria-current="page" @endif class="flex items-center gap-2.5 text-base font-semibold text-ink hover:text-primary">
                        <span aria-hidden="true" @class([
                            'flex size-4.5 shrink-0 items-center justify-center rounded-[5px] border-[1.5px]',
                            'border-primary bg-primary text-white' => $item['active'],
                            'border-muted/50 bg-surface' => ! $item['active'],
                        ])>@if ($item['active'])<x-icon name="check" class="size-3"/>@endif</span>
                        {{ $item['label'] }}
                        @if ($item['count'] !== null)<span class="ms-auto text-[12.5px] text-muted">{{ $item['count'] }}</span>@endif
                    </a></li>
                @endforeach
            </ul>
        </nav>
    @endif

    <form method="get" action="{{ $filters['action'] }}" aria-label="{{ __('shop.filters.form') }}" class="flex flex-col">
        @if ($filters['sizes'] !== [])
            <fieldset class="{{ $block }}">
                <legend class="{{ $legend }}">{{ __('shop.filters.sizes') }}</legend>
                <div class="flex flex-wrap gap-2">
                    @foreach ($filters['sizes'] as $size)
                        <label class="{{ $pill }}"><input type="checkbox" name="size[]" value="{{ $size['value'] }}" @checked($size['checked']) class="sr-only">{{ $size['value'] }}</label>
                    @endforeach
                </div>
            </fieldset>
        @endif

        @if ($filters['colors'] !== [])
            <fieldset class="{{ $block }}">
                <legend class="{{ $legend }}">{{ __('shop.filters.colors') }}</legend>
                <div class="flex flex-wrap gap-2">
                    @foreach ($filters['colors'] as $color)
                        <label title="{{ $color['value'] }}" class="flex size-8.5 cursor-pointer items-center justify-center rounded-full border-2 border-line hover:border-primary has-checked:border-primary has-focus-visible:outline-2 has-focus-visible:outline-primary">
                            <input type="checkbox" name="color[]" value="{{ $color['value'] }}" @checked($color['checked']) class="sr-only">
                            <span class="sr-only">{{ $color['label'] }}</span>
                            <svg viewBox="0 0 30 30" aria-hidden="true" class="size-full"><circle cx="15" cy="15" r="15" @if ($color['hex']) fill="{{ $color['hex'] }}" @else class="fill-line" @endif/></svg>
                        </label>
                    @endforeach
                </div>
            </fieldset>
        @endif

        <fieldset class="{{ $block }}">
            <legend class="{{ $legend }}">{{ __('shop.filters.price') }}</legend>
            <div class="flex gap-2.5">
                <label class="flex flex-1 flex-col gap-1 text-[12.5px] font-bold text-muted">{{ __('shop.filters.price_min') }}
                    <input type="text" inputmode="numeric" name="min" value="{{ $filters['price']['min'] }}" autocomplete="off" @if ($filters['price']['from']) placeholder="{{ $filters['price']['from'] }}" @endif
                           class="h-10 w-full rounded-2xl border-[1.5px] border-line bg-surface px-3 text-base font-semibold text-ink focus:border-primary">
                </label>
                <label class="flex flex-1 flex-col gap-1 text-[12.5px] font-bold text-muted">{{ __('shop.filters.price_max') }}
                    <input type="text" inputmode="numeric" name="max" value="{{ $filters['price']['max'] }}" autocomplete="off" @if ($filters['price']['to']) placeholder="{{ $filters['price']['to'] }}" @endif
                           class="h-10 w-full rounded-2xl border-[1.5px] border-line bg-surface px-3 text-base font-semibold text-ink focus:border-primary">
                </label>
            </div>
            @if ($filters['price']['from'] && $filters['price']['to'])
                <p class="m-0 flex justify-between text-sm font-bold text-muted"><span class="sr-only">{{ __('shop.filters.price_range', ['min' => $filters['price']['from'], 'max' => $filters['price']['to']]) }}</span><span aria-hidden="true">{{ $filters['price']['from'] }}</span><span aria-hidden="true">{{ $filters['price']['to'] }}</span></p>
            @endif
        </fieldset>

        @if ($filters['brands'] !== [])
            <fieldset class="{{ $block }}">
                <legend class="{{ $legend }}">{{ __('shop.filters.brands') }}</legend>
                <div class="flex flex-col gap-3">
                    @foreach ($filters['brands'] as $brand)
                        <label class="flex cursor-pointer items-center gap-2.5 text-base font-semibold text-ink">
                            <input type="checkbox" name="brand[]" value="{{ $brand['value'] }}" @checked($brand['checked']) class="size-4.5 shrink-0 accent-primary">
                            {{ $brand['label'] }}<span class="ms-auto text-[12.5px] text-muted">{{ $brand['count'] }}</span>
                        </label>
                    @endforeach
                </div>
            </fieldset>
        @endif

        <div class="{{ $block }}">
            <label class="group flex cursor-pointer items-center justify-between gap-3 text-base font-bold text-ink">
                {{ __('shop.filters.stock') }}
                <input type="checkbox" name="stock" value="1" @checked($filters['stock']) class="sr-only">
                <span aria-hidden="true" class="relative h-8 w-13 shrink-0 rounded-full bg-line transition-colors group-has-checked:bg-primary group-has-focus-visible:outline-2 group-has-focus-visible:outline-primary">
                    <span class="absolute start-1 top-1 size-6 rounded-full bg-surface transition-all group-has-checked:start-6"></span>
                </span>
            </label>
        </div>

        @foreach ($filters['hidden'] as $name => $value)
            <input type="hidden" name="{{ $name }}" value="{{ $value }}">
        @endforeach
        <x-ui.button type="submit" size="md" class="mt-2 justify-center">{{ __('shop.filters.apply') }}</x-ui.button>
    </form>
</section>
