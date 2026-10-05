{{--
    <x-ui.eyebrow tone="light|dark" color="primary|cycle" pill icon="sprout">ابزارهای این مرحله</x-ui.eyebrow>
    Small label above a heading (14px/800, AUDIT §2.2). `pill`: the hero variant (night-card/80 fill + lilac/35 border on
    night; L9-03: the design's lilac/14 fill over the hero glow left the lilac text at 3.4:1, now ≥ 6:1). `color="cycle"` is the promise banner's stage-red label. Not a heading — the h1/h2 follows it.
--}}
@props(['tone' => 'light', 'color' => 'primary', 'pill' => false, 'icon' => null])
@php
    $colorClass = match (true) {
        $tone === 'dark' => 'text-lilac',
        $color === 'cycle' => 'text-stage-cycle',
        default => 'text-primary',
    };
    $pillClasses = $pill
        ? ($tone === 'dark'
            ? 'self-start inline-flex items-center gap-2 rounded-3xl border border-lilac/35 bg-night-card/80 px-4 py-2'
            : 'self-start inline-flex items-center gap-2 rounded-3xl border border-primary/35 bg-primary/10 px-4 py-2')
        : '';
@endphp
<span {{ $attributes->class("text-base font-extrabold {$colorClass} {$pillClasses}") }}>@if ($icon)<x-icon :name="$icon" class="size-4"/>@endif{{ $slot }}</span>
