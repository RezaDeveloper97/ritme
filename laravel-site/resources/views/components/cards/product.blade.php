{{--
    <x-cards.product href="…" title="بادی آستین‌بلند نخی · ۳ عدد" seller="پوشاک پنبه‌ریز" :rating="4.7" :reviews="214"
                     :price="485000" :compare="520000" badge="پنبه ۱۰۰٪" :media="$mediaId" illustration="product-bodysuit" tint="pregnancy"/>
    Product card markup (AUDIT §2.3; data from L6-02): 220px rounded-24 image tile (<x-picture>, or the design's
    illustration on a stage tint) + corner badge + wishlist heart, title 15 (h3), seller 12.5, rating, price + compare.
    The title link is stretched over the card; the heart is a separate button above it (no nested interactive
    elements). The heart is inert until L6 wires it (`wishlist` = false hides it).
--}}
@props([
    'href',
    'title',
    'seller' => null,
    'rating' => null,
    'reviews' => null,
    'price' => null,
    'compare' => null,
    'badge' => null,
    'media' => null,
    'illustration' => null,
    'tint' => 'primary',
    'wishlist' => true,
    'as' => 'h3',
])
@php
    $tintClass = match ($tint) {
        'cycle' => 'bg-stage-cycle/15',
        'ttc' => 'bg-stage-ttc/15',
        'pregnancy' => 'bg-stage-pregnancy/15',
        'postpartum' => 'bg-stage-postpartum/15',
        'teen' => 'bg-stage-teen/15',
        'lavender' => 'bg-lavender',
        default => 'bg-primary/12',
    };
@endphp
<article {{ $attributes->class('relative flex flex-col gap-2.5 text-ink') }}>
    <div class="relative">
        <div class="relative h-55 overflow-hidden rounded-4xl {{ $tintClass }}">
            @if ($media)
                <x-picture :media="$media" :alt="$title" sizes="(max-width: 700px) 50vw, 20vw" class="size-full object-cover"/>
            @elseif ($illustration)
                <x-illustration :name="$illustration" class="size-full"/>
            @endif
            @if ($badge)<x-ui.pill tone="surface" size="xs" class="absolute start-2 top-2">{{ $badge }}</x-ui.pill>@endif
        </div>
        @if ($wishlist)
            <button type="button" aria-label="افزودن {{ $title }} به علاقه‌مندی‌ها" aria-pressed="false" class="absolute end-2.5 bottom-2.5 z-10 flex size-10 items-center justify-center rounded-full bg-surface text-ink hover:text-primary">
                <x-icon name="heart" class="size-4.5"/>
            </button>
        @endif
    </div>
    <div class="flex flex-col gap-1">
        <{{ $as }} class="m-0 text-md leading-normal font-bold"><a href="{{ $href }}" class="text-ink after:absolute after:inset-0 hover:text-primary">{{ $title }}</a></{{ $as }}>
        @if ($seller)<span class="text-[12.5px] font-semibold text-muted">{{ $seller }}</span>@endif
        @if ($rating !== null)<x-ui.rating :value="$rating" :count="$reviews"/>@endif
        @if ($price !== null)<x-ui.price :amount="$price" :compare="$compare"/>@endif
    </div>
</article>
