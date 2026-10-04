{{--
    <x-cards.review name="مادر پرنیا" :rating="5" meta="خرید تأییدشده · سایز ۳-۶ ماه" text="…"/>                 (shop-product)
    <x-cards.review avatar name="مادر رها" :rating="5" meta="۲ هفته پیش · رزرو از ریتمی" text="…"/>             (directory-place)
    Review card (AUDIT §2.3): radius 28 (24 with avatar), p 20; name 15 + star score; meta 12.5 muted; text 14.5 lh 2.
    `avatar` shows the first letter in a 42px primary/13 circle. Meta strings (Jalali / relative dates) come ready-made.
--}}
@props(['name', 'rating' => null, 'meta' => null, 'text', 'avatar' => false])
<article {{ $attributes->class(['flex flex-col bg-surface p-5 border border-line', $avatar ? 'gap-2.5 rounded-4xl' : 'rounded-5xl']) }}>
    @if ($avatar)
        <header class="flex items-center gap-3">
            <span aria-hidden="true" class="flex size-10.5 shrink-0 items-center justify-center rounded-full border border-primary/33 bg-primary/13 text-lg font-extrabold text-primary">{{ mb_substr(trim(preg_replace('/^مادر\s+/u', '', $name) ?? $name), 0, 1) }}</span>
            <div class="grow">
                <b class="text-md">{{ $name }}</b>
                @if ($meta)<div class="text-[12.5px] font-semibold text-muted">{{ $meta }}</div>@endif
            </div>
            @if ($rating !== null)<x-ui.rating :value="$rating" size="xs"/>@endif
        </header>
    @else
        <header class="flex items-center justify-between">
            <b class="text-md">{{ $name }}</b>
            @if ($rating !== null)<x-ui.rating :value="$rating" size="xs"/>@endif
        </header>
        @if ($meta)<div class="mt-1 mb-2 text-[12.5px] font-semibold text-muted">{{ $meta }}</div>@endif
    @endif
    <p class="m-0 text-[14.5px] leading-loose">{{ $text }}</p>
</article>
