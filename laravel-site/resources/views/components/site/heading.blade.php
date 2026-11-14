{{--
    <x-site.heading id="cycle-features-title" eyebrow="امکانات ریتمی" eyebrow-color="text-stage-cycle" title="…"/>
    <x-site.heading … align="center" lead="…"/>       <x-site.heading … :more-href="$url" more-label="همه مقاله‌ها"/>
    Section heading of the Nov 2026 redesign: eyebrow 14/800 (primary by default, a stage colour on stage pages),
    h2 Lalezar 40 lh 1.25, optional lead 17 lh 2 muted, optional underlined «more» link at the end of the row.
    `level` switches the heading tag (h1 on pages whose first section is a heading).
--}}
@props([
    'title',
    'id' => null,
    'eyebrow' => null,
    'eyebrowColor' => 'text-primary',
    'lead' => null,
    'align' => 'start',
    'moreHref' => null,
    'moreLabel' => null,
    'level' => 'h2',
    'size' => 'text-d-lg',
])
@php($center = $align === 'center')
<div {{ $attributes->class(['flex items-end justify-between gap-6 max-sm:flex-col max-sm:items-start' => $moreHref, 'flex flex-col gap-3' => ! $moreHref, 'items-center text-center' => $center && ! $moreHref]) }}>
    <div @class(['flex flex-col gap-3', 'items-center text-center' => $center])>
        @if ($eyebrow)<span class="text-base font-extrabold {{ $eyebrowColor }}">{{ $eyebrow }}</span>@endif
        <{{ $level }} @if ($id) id="{{ $id }}" @endif class="m-0 font-display {{ $size }} leading-[1.25] font-normal text-ink">{{ $title }}</{{ $level }}>
        @if ($lead)<p @class(['m-0 max-w-160 text-xl leading-loose font-medium text-muted', 'mx-auto' => $center])>{{ $lead }}</p>@endif
    </div>
    @if ($moreHref)
        <x-site.text-link :href="$moreHref">{{ $moreLabel }}</x-site.text-link>
    @endif
</div>
