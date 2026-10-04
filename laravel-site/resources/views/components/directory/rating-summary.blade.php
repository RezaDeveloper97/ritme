{{--
    <x-directory.rating-summary :rating="['average' => 4.8, 'count' => 126, 'aspects' => [['label' => 'تمیزی', 'value' => 4.7, 'percent' => 95]]]"/>
    Score block of «نظر مادرها» (L5-03): big Lalezar average, «از N نظر», one bar per scored aspect (widths in 5%
    steps → `w-[n%]` utilities generated below). Only real approved reviews count; null renders the empty state.
--}}
@props(['rating' => null])
@php
    // Literal class names so Tailwind generates them (5% steps).
    $widths = ['w-[0%]', 'w-[5%]', 'w-[10%]', 'w-[15%]', 'w-[20%]', 'w-[25%]', 'w-[30%]', 'w-[35%]', 'w-[40%]', 'w-[45%]', 'w-[50%]',
        'w-[55%]', 'w-[60%]', 'w-[65%]', 'w-[70%]', 'w-[75%]', 'w-[80%]', 'w-[85%]', 'w-[90%]', 'w-[95%]', 'w-[100%]'];
@endphp
@if ($rating === null)
    <p {{ $attributes->class('m-0 flex items-center gap-3 rounded-3xl border border-line bg-surface px-5 py-4 text-base font-semibold text-muted') }}>
        <x-icon name="star" class="size-5 shrink-0 text-stage-ttc"/>{{ __('directory.place.no_rating') }}
    </p>
@else
    <div {{ $attributes->class('flex items-center gap-8 max-lg:flex-wrap') }}>
        <div class="flex flex-col items-center">
            <span class="font-display text-[56px] leading-heading max-lg:text-[42px] max-sm:text-[34px]">{{ \App\Support\Text\PersianDigits::number($rating['average'], 1) }}</span>
            <span class="text-sm font-bold text-muted">{{ __('directory.place.reviews.from', ['count' => \App\Support\Text\PersianDigits::number($rating['count'])]) }}</span>
        </div>
        @if ($rating['aspects'] !== [])
            <dl class="m-0 flex flex-1 flex-col gap-2.5 max-lg:basis-75 max-sm:basis-full">
                @foreach ($rating['aspects'] as $aspect)
                    <div class="flex items-center gap-3 text-sm-plus font-bold">
                        <dt class="w-17.5 text-muted">{{ $aspect['label'] }}</dt>
                        <dd class="m-0 flex grow items-center gap-3">
                            <span aria-hidden="true" class="flex h-2 grow overflow-hidden rounded-full bg-line"><span class="{{ $widths[intdiv(max(0, min(100, $aspect['percent'])), 5)] }} bg-stage-ttc"></span></span>
                            <span class="w-7.5">{{ \App\Support\Text\PersianDigits::number($aspect['value'], 1) }}</span>
                        </dd>
                    </div>
                @endforeach
            </dl>
        @endif
    </div>
@endif
