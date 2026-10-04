{{--
    Place page (L5-03, design/html/directory-place.html): /directory/place/{slug}.
    Data (ShowPlaceController, arrays + scalars only): $place (header), $breadcrumbs (list<BreadcrumbItem>), $gallery,
    $about, $amenities, $services, $hours, $note, $rating, $reviews, $reviewPagination, $reviewForm, $address, $rules,
    $booking. JSON-LD (ItemPage, LocalBusiness subtype, BreadcrumbList via <x-ui.breadcrumbs>) is registered before the
    head renders. No embedded map (address illustration + deep links). JS: `gallery` (lightbox) and `share` modules.
--}}
@extends('layouts.app')

@section('content')
    <div class="flex flex-col gap-7 px-30 pt-6 pb-20 max-lg:px-5 max-lg:pb-11">
        <x-ui.breadcrumbs :items="$breadcrumbs"/>

        <x-directory.header :place="$place"/>

        <x-directory.gallery :images="$gallery['images']" :full="$gallery['full']" :illustrations="$gallery['illustrations']" :name="$gallery['alt']"/>

        <div class="flex items-start gap-14 max-lg:flex-wrap">
            <div class="flex min-w-0 flex-1 flex-col max-lg:basis-75 max-sm:basis-full">
                @if ($about)
                    <x-directory.section id="about" :title="__('directory.place.about')">
                        <p class="m-0 text-lg leading-loose font-medium text-muted">{{ $about }}</p>
                    </x-directory.section>
                @endif

                @if ($amenities !== [])
                    <x-directory.section id="amenities" :title="__('directory.place.amenities')">
                        <ul class="m-0 grid list-none grid-cols-4 gap-4 p-0 max-lg:grid-cols-2 max-sm:grid-cols-1">
                            @foreach ($amenities as $amenity)
                                <li class="flex items-center gap-2.5 text-[14.5px] font-bold">
                                    <span aria-hidden="true" class="flex size-10 shrink-0 items-center justify-center rounded-full bg-primary/13"><x-icon :name="$amenity['icon']" class="size-5 text-primary"/></span>{{ $amenity['label'] }}
                                </li>
                            @endforeach
                        </ul>
                    </x-directory.section>
                @endif

                @if ($services !== [])
                    <x-directory.section id="services" :title="__('directory.place.services.title')" :lead="__('directory.place.services.lead')">
                        <ul class="m-0 flex list-none flex-col gap-2.5 p-0">
                            @foreach ($services as $service)
                                <li class="flex items-center gap-4 rounded-3xl border-[1.5px] border-line bg-surface px-5 py-4.5 max-lg:flex-wrap">
                                    <div class="grow">
                                        <h3 class="m-0 text-lg font-bold">{{ $service['name'] }}</h3>
                                        @if ($service['meta'])<p class="mt-0.5 mb-0 text-sm-plus font-semibold text-muted">{{ $service['meta'] }}</p>@endif
                                    </div>
                                    @if ($service['price'])
                                        <b class="text-lg">{{ \App\Support\Text\Toman::withUnit($service['price']) }}</b>
                                    @else
                                        <span class="text-sm-plus font-semibold text-muted">{{ __('directory.place.services.ask') }}</span>
                                    @endif
                                    <a href="#book" class="box-content flex h-10 items-center rounded-full border-[1.5px] border-primary px-4 text-sm-plus font-extrabold text-ink hover:bg-primary/11 hover:text-ink"
                                       aria-label="{{ __('directory.place.services.choose_label', ['name' => $service['name']]) }}">{{ __('directory.place.services.choose') }}</a>
                                </li>
                            @endforeach
                        </ul>
                    </x-directory.section>
                @endif

                @if ($hours !== [])
                    <x-directory.section id="hours" :title="__('directory.place.hours.title')">
                        <x-directory.hours :rows="$hours"/>
                    </x-directory.section>
                @endif

                @if ($note)
                    <p class="m-0 flex items-start gap-3 rounded-3xl border border-line bg-surface px-5 py-4 text-base leading-relaxed font-semibold text-muted">
                        <x-icon name="info" class="size-5 shrink-0 text-stage-postpartum"/>{{ $note }}
                    </p>
                @endif

                <x-directory.section id="reviews" :title="__('directory.place.reviews.title')" :lead="__('directory.place.reviews.lead')">
                    <x-directory.rating-summary :rating="$rating"/>
                    @if ($reviews !== [])
                        <div class="grid grid-cols-2 gap-3.5 max-sm:grid-cols-1">
                            @foreach ($reviews as $review)
                                <x-cards.review avatar :name="$review['name']" :rating="$review['rating']" :meta="$review['meta']" :text="$review['text']"/>
                            @endforeach
                        </div>
                        @if (collect($reviews)->contains('demo', true))
                            <p class="m-0 text-sm font-semibold text-muted">{{ __('directory.place.reviews.demo_note') }}</p>
                        @endif
                    @else
                        <p class="m-0 text-base font-semibold text-muted">{{ __('directory.place.reviews.empty') }}</p>
                    @endif
                    @if ($reviewPagination)
                        <nav aria-label="{{ __('directory.place.reviews.pagination') }}" class="flex items-center justify-between gap-3 text-sm-plus font-extrabold">
                            @if ($reviewPagination['previous'])<a href="{{ $reviewPagination['previous'] }}" rel="prev" class="text-primary">{{ __('directory.place.reviews.previous') }}</a>@else<span></span>@endif
                            <span class="text-muted">{{ $reviewPagination['label'] }}</span>
                            @if ($reviewPagination['next'])<a href="{{ $reviewPagination['next'] }}" rel="next" class="text-primary">{{ __('directory.place.reviews.next') }}</a>@else<span></span>@endif
                        </nav>
                    @endif
                    <x-directory.review-form :action="$reviewForm['action']" :aspects="$reviewForm['aspects']" :honeypot="$reviewForm['honeypot']"/>
                </x-directory.section>

                <x-directory.section id="address" :title="__('directory.place.address.title')">
                    <x-directory.address :line="$address['line']" :links="$address['links']" :pin="$address['pin']"/>
                </x-directory.section>

                @if ($rules !== [])
                    <x-directory.section id="rules" :title="__('directory.place.rules')">
                        <ul class="m-0 flex list-none flex-col gap-1.5 p-0">
                            @foreach ($rules as $rule)
                                <li class="flex items-start gap-2.5 text-lg leading-relaxed font-semibold text-ink">
                                    <x-icon name="check" class="mt-1.5 size-4.5 shrink-0 text-stage-teen"/>{{ $rule }}
                                </li>
                            @endforeach
                        </ul>
                    </x-directory.section>
                @endif

                <div class="flex items-center justify-between rounded-4xl bg-lavender px-7 py-6 max-lg:flex-wrap">
                    <span class="flex items-center gap-2.5 text-md font-bold"><x-icon name="store" class="size-5.5 text-primary"/>{{ __('directory.place.owner.title') }}</span>
                    <span class="flex gap-4.5 max-lg:flex-wrap">
                        <a href="{{ route('directory.business') }}" class="text-base font-extrabold">{{ __('directory.place.owner.manage') }}</a>
                        <a href="{{ route('contact') }}" class="text-base font-bold text-muted">{{ __('directory.place.owner.report') }}</a>
                    </span>
                </div>
            </div>

            <div class="pt-8 max-sm:w-full">
                <x-directory.booking :price-from="$booking['priceFrom']" :unit="$booking['priceUnit']" :phone="$booking['phone']"/>
            </div>
        </div>
    </div>
@endsection
