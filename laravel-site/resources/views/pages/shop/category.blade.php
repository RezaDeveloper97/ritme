{{--
    Shop category listing (L6-02, design/html/shop-list.html): /shop/category/{slug} with crawlable GET filters.
    Data (Shop\CategoryController): $category (CategoryData), $breadcrumbs (list<BreadcrumbItem>, also the JSON-LD
    BreadcrumbList via x-ui.breadcrumbs), $subnav, $summary, $sorts (chip links), $filters (side panel), $active
    (removable filter chips), $cards (list<ProductCardData>), $media (cover MediaData by id), $demo, $empty,
    $resetUrl, $pagination. CollectionPage + ItemList are registered by the controller. No JS beyond the cart badge.
--}}
@extends('layouts.app', ['navRoute' => 'shop.index'])

@section('content')
    <x-shop.subnav :departments="$subnav['departments']" :links="$subnav['links']" :label="$subnav['label']"/>

    <section aria-labelledby="results-title" class="flex flex-col gap-6 px-30 pt-7 pb-20 max-lg:px-5 max-lg:pb-11">
        <x-ui.breadcrumbs :items="$breadcrumbs"/>

        <div class="flex items-end justify-between gap-4 max-lg:flex-wrap">
            <div>
                <h1 class="m-0 font-display text-d-xl leading-display font-normal text-ink">{{ $category->name }}</h1>
                <p class="m-0 mt-1.5 text-[14.5px] font-semibold text-muted">{{ $summary }}</p>
            </div>
            <div class="flex items-center gap-2 text-base font-bold text-muted max-lg:flex-wrap">
                <span aria-hidden="true">{{ __('shop.sort.label') }}</span>
                <x-ui.chip-nav size="sm" current="true" :label="__('shop.sort.menu')" :items="$sorts"/>
            </div>
        </div>

        <div class="flex items-start gap-10 max-lg:flex-wrap">
            <x-shop.filters :filters="$filters" :reset-url="$resetUrl"/>

            <div class="flex min-w-0 flex-1 flex-col gap-8 max-lg:basis-75 max-sm:basis-full">
                <h2 id="results-title" class="sr-only">{{ __('shop.category.results', ['name' => $category->name]) }}</h2>

                @if ($active !== [])
                    <ul aria-label="{{ __('shop.filters.active') }}" class="m-0 flex list-none flex-wrap gap-2 p-0">
                        @foreach ($active as $chip)
                            <li><a href="{{ $chip['href'] }}" aria-label="{{ __('shop.filters.remove', ['name' => $chip['label']]) }}" class="flex h-9 items-center gap-1.5 rounded-full bg-lavender px-3 text-sm font-bold text-ink hover:text-primary">{{ $chip['label'] }}<x-icon name="x" class="size-3.5 text-muted"/></a></li>
                        @endforeach
                    </ul>
                @endif

                @if ($demo)
                    <p class="m-0 -mb-4 flex items-center gap-2 text-sm font-semibold text-muted"><x-icon name="info" class="size-4 shrink-0 text-primary"/>{{ __('shop.demo_note') }}</p>
                @endif

                @if ($cards === [])
                    <div class="flex flex-col items-start gap-4 rounded-4xl border border-line bg-surface p-8">
                        <p class="m-0 text-lg leading-loose font-medium text-muted">{{ $empty }}</p>
                        @if ($filters['filtered'])
                            <x-ui.button :href="$resetUrl" variant="outline" size="md">{{ __('shop.category.reset') }}</x-ui.button>
                        @endif
                    </div>
                @else
                    <div class="grid grid-cols-4 gap-x-5 gap-y-7 max-lg:grid-cols-2 max-sm:grid-cols-1">
                        @foreach ($cards as $product)
                            <x-shop.product-card :product="$product" :media="$media" :index="$loop->index" class="[&>div>div:first-child]:h-57.5"/>
                        @endforeach
                    </div>
                @endif

                @if ($pagination)
                    <x-shop.pagination :pagination="$pagination"/>
                @endif
            </div>
        </div>
    </section>
@endsection
