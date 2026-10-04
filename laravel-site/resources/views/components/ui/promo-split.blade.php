{{--
    <x-ui.promo-split :items="[
        ['href' => route('shop.category', 'baby-clothes'), 'tone' => 'lavender', 'eyebrow' => 'سیسمونی و نوزاد',
         'title' => 'برای آمدن نوزاد آماده شو', 'text' => '…', 'cta' => 'ورود به فروشگاه', 'illustration' => 'product-bodysuit'],
        ['href' => route('shop.index'), 'tone' => 'dashed', 'icon' => 'store', 'title' => 'فروشگاه', 'text' => '…', 'cta' => 'ورود به فروشگاه'],
    ]"/>
    Two large promo cards side by side (AUDIT §2.4: shop, services). Tones: lavender | blush (danger-soft) | dashed
    (white, dashed border). With `illustration` (an <x-illustration> name) the card is the shop's horizontal layout with
    an ink pill CTA; otherwise the services' vertical layout with an icon and a text link. Titles are h2 (`level`).
--}}
@props(['items' => [], 'level' => 'h2'])
<div {{ $attributes->class('flex gap-6 max-lg:flex-wrap') }}>
    @foreach ($items as $item)
        @php
            $toneClasses = match ($item['tone'] ?? 'lavender') {
                'blush' => 'bg-danger-soft',
                'dashed' => 'border-[1.5px] border-dashed border-line bg-surface',
                default => 'bg-lavender',
            };
            $wide = ! empty($item['illustration']);
            $dashed = ($item['tone'] ?? '') === 'dashed';
            // The design is content-box: its ≤1024 `flex-basis:300px` excludes padding (+ the dashed border), so the
            // border-box basis is 300 + 2×padding (+ 2×1.5px border). Keeps the design's stacking at tablet widths.
            $basis = match (true) {
                $wide && $dashed => 'max-lg:basis-[383px]',
                $wide => 'max-lg:basis-95',
                $dashed => 'max-lg:basis-[375px]',
                default => 'max-lg:basis-93',
            };
        @endphp
        <a href="{{ $item['href'] }}" @class([
            'flex flex-1 overflow-hidden text-ink transition-shadow hover:text-ink hover:shadow-card max-sm:basis-full',
            $basis,
            $toneClasses,
            'items-center gap-6 rounded-[36px] p-10 max-lg:flex-wrap max-sm:p-6' => $wide,
            'flex-col gap-3.5 rounded-6xl p-9 max-sm:p-6' => ! $wide,
        ])>
            @if ($wide)
                <span class="flex flex-1 flex-col gap-3.5 max-lg:basis-75 max-sm:basis-full">
                    @if (! empty($item['eyebrow']))<x-ui.eyebrow>{{ $item['eyebrow'] }}</x-ui.eyebrow>@endif
                    <{{ $level }} class="m-0 font-display text-[38px] leading-heading font-normal text-ink">{{ $item['title'] }}</{{ $level }}>
                    @if (! empty($item['text']))<span class="text-[15.5px] leading-loose font-medium text-muted">{{ $item['text'] }}</span>@endif
                    @if (! empty($item['cta']))<span class="flex h-12 items-center gap-2 self-start rounded-full bg-ink px-5.5 text-[14.5px] font-extrabold text-white">{{ $item['cta'] }}<x-icon name="arrow-left" class="size-4"/></span>@endif
                </span>
                <span aria-hidden="true" class="relative size-55 shrink-0 overflow-hidden rounded-5xl bg-surface/60"><x-illustration :name="$item['illustration']" class="size-full"/></span>
            @else
                @if (! empty($item['icon']))<x-icon :name="$item['icon']" stroke="1.8" @class(['size-10', ($item['tone'] ?? '') === 'dashed' ? 'text-muted' : 'text-primary'])/>@endif
                @if (! empty($item['eyebrow']))<x-ui.eyebrow>{{ $item['eyebrow'] }}</x-ui.eyebrow>@endif
                <{{ $level }} class="m-0 font-display text-d-md leading-heading font-normal text-ink">{{ $item['title'] }}</{{ $level }}>
                @if (! empty($item['text']))<span class="text-lg leading-loose font-medium text-muted">{{ $item['text'] }}</span>@endif
                @if (! empty($item['cta']))<span class="flex items-center gap-1.5 text-md font-extrabold text-primary">{{ $item['cta'] }}<x-icon name="arrow-left" class="size-4"/></span>@endif
            @endif
        </a>
    @endforeach
</div>
