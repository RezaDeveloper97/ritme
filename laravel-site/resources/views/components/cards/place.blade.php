{{--
    <x-cards.place href="…" name="استخر مادر و کودک آب‌پری" :rating="4.8" :reviews="126" category="استخر مادر و کودک"
                   district="ونک" distance="۱٫۲ کیلومتر" ages="مناسب ۶ ماه تا ۴ سال" :price-from="320000"
                   :slots="['امروز ۱۷:۰۰']" verified :media="$coverId" illustration="place-cover-pool" selected/>
    Directory place card markup (AUDIT §2.3; data from L5-02): radius 24, 190px cover (<x-picture> or illustration)
    + verified pill + bookmark button; name 17 (h3, stretched link) + rating; category · district · distance;
    ages · «از … تومان»; next-slot chips. `selected` = 2px primary border (map selection).
--}}
@props([
    'href',
    'name',
    'rating' => null,
    'reviews' => null,
    'category' => null,
    'district' => null,
    'distance' => null,
    'ages' => null,
    'priceFrom' => null,
    'slots' => [],
    'verified' => false,
    'media' => null,
    'illustration' => null,
    'selected' => false,
    'bookmark' => true,
    'as' => 'h3',
])
@php($facts = array_values(array_filter([$category, $district, $distance], static fn ($v): bool => $v !== null && $v !== '')))
<article {{ $attributes->class(['relative flex flex-col overflow-hidden rounded-4xl bg-surface text-ink', $selected ? 'border-2 border-primary' : 'border border-line']) }}>
    <div class="relative h-47.5 overflow-hidden bg-lavender">
        @if ($media)
            <x-picture :media="$media" :alt="$name" sizes="(max-width: 700px) 100vw, 33vw" class="size-full object-cover"/>
        @elseif ($illustration)
            <x-illustration :name="$illustration" class="size-full"/>
        @endif
        @if ($verified)<x-ui.pill tone="surface" size="md" icon="shield-check" icon-class="text-stage-teen" class="absolute start-3 top-3">مدارک بررسی شد</x-ui.pill>@endif
        @if ($bookmark)
            <button type="button" aria-label="ذخیره {{ $name }}" aria-pressed="false" class="absolute end-3 top-3 z-10 flex size-10 items-center justify-center rounded-full bg-surface/92 text-ink hover:text-primary">
                <x-icon name="bookmark" class="size-4.5"/>
            </button>
        @endif
    </div>
    <div class="flex flex-col gap-2 px-4.5 pt-4 pb-4.5">
        <div class="flex items-center justify-between gap-2 max-lg:flex-wrap">
            <{{ $as }} class="m-0 text-xl font-bold"><a href="{{ $href }}" class="text-ink after:absolute after:inset-0 hover:text-primary">{{ $name }}</a></{{ $as }}>
            @if ($rating !== null)<x-ui.rating :value="$rating" :count="$reviews" size="md"/>@endif
        </div>
        @if ($facts !== [])
            <div class="flex items-center gap-2 text-sm-plus font-semibold text-muted max-lg:flex-wrap">
                @foreach ($facts as $fact)
                    @unless ($loop->first)<span aria-hidden="true" class="inline-block size-[3px] rounded-full bg-muted"></span><span class="sr-only">·</span>@endunless{{ $fact }}
                @endforeach
            </div>
        @endif
        @if ($ages || $priceFrom !== null || $slots !== [])
            <div class="mt-1 flex items-center justify-between gap-2 max-lg:flex-wrap">
                <span class="text-sm-plus font-semibold text-muted">{{ $ages }}@if ($ages && $priceFrom !== null) · @endif @if ($priceFrom !== null)<x-ui.price :amount="$priceFrom" from size="inline"/>@endif</span>
                @foreach ($slots as $nextSlot)
                    <x-ui.pill dot>{{ $nextSlot }}</x-ui.pill>
                @endforeach
            </div>
        @endif
    </div>
</article>
