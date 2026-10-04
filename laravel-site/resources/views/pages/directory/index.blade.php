{{--
    Directory listing (L5-02, design/html/directory.html): /directory, /directory/{city}, /directory/{city}/{category}.
    Data (ListPlacesController, arrays + DTO-derived scalars only): $intro [eyebrow, title, lead], $breadcrumbs
    (?list<BreadcrumbItem>, landings only), $search, $chips, $filters, $sorts, $summary, $cards (x-cards.place props),
    $pagination, $resetUrl, $areas. JSON-LD (CollectionPage, ItemList, BreadcrumbList) is registered by the controller.
    The design's map column is replaced by a static illustration + crawlable city / district / landing links (no maps,
    AUDIT §8). No JS: filters are GET links and forms, the panels are native <details>.
--}}
@extends('layouts.app')

@section('content')
    @include('pages.directory.partials.intro')

    <section aria-labelledby="results-title" class="flex items-start gap-8 px-30 pt-2 pb-16 max-lg:flex-wrap max-lg:px-5 max-lg:pb-[35.2px]">
        <div class="flex min-w-0 flex-1 flex-col gap-5 max-lg:basis-75 max-sm:basis-full">
            <h2 id="results-title" class="sr-only">{{ __('directory.results.heading') }}</h2>
            @include('pages.directory.partials.toolbar')

            @if ($cards === [])
                <div class="flex flex-col items-start gap-4 rounded-4xl border border-line bg-surface p-8">
                    <p class="m-0 text-lg leading-loose font-medium text-muted">{{ __('directory.results.empty') }}</p>
                    <x-ui.button :href="$resetUrl" variant="outline" size="md">{{ __('directory.results.reset') }}</x-ui.button>
                </div>
            @else
                <div class="grid grid-cols-2 gap-5 max-sm:grid-cols-1">
                    @foreach ($cards as $card)
                        <x-cards.place :href="$card['href']" :name="$card['name']" :rating="$card['rating']" :reviews="$card['reviews']"
                                       :category="$card['category']" :district="$card['district']" :distance="$card['distance']"
                                       :ages="$card['ages']" :price-from="$card['priceFrom']" :slots="$card['slots']"
                                       :verified="$card['verified']" :media="$card['media']" :illustration="$card['illustration']"/>
                    @endforeach
                </div>
            @endif

            @if ($pagination)
                @include('pages.directory.partials.pagination')
            @endif

            <p class="m-0 flex items-center gap-3 rounded-3xl border border-line bg-surface px-5 py-4 text-base leading-relaxed font-semibold text-muted">
                <x-icon name="shield-check" class="size-5.5 shrink-0 text-stage-teen"/>{{ __('directory.fair') }}
            </p>
        </div>

        @include('pages.directory.partials.areas')
    </section>

    @include('pages.directory.partials.business')
@endsection
