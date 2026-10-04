{{--
    <x-ui.section pad="xl|lg|md|sm|none" bg="canvas|surface|night|none" as="section" aria-labelledby="…">…</x-ui.section>
    Full-width band with the page gutters. Vertical paddings 96 / 80 / 64·32 / 32 (desktop) shrink ×0.55 ≤1024
    (AUDIT §3.4); horizontal 120 → 20.
--}}
@props(['pad' => 'xl', 'bg' => 'none', 'as' => 'section'])
@php
    $padClasses = match ($pad) {
        'lg' => 'py-20 max-lg:py-11',
        'md' => 'pt-16 pb-8 max-lg:pt-[35.2px]',
        'sm' => 'py-8',
        'none' => '',
        default => 'py-24 max-lg:py-[52.8px]',
    };
    $bgClasses = match ($bg) {
        'canvas' => 'bg-canvas',
        'surface' => 'bg-surface',
        'night' => 'bg-night text-on-night',
        default => '',
    };
@endphp
<{{ $as }} {{ $attributes->class("px-30 max-lg:px-5 {$padClasses} {$bgClasses}") }}>{{ $slot }}</{{ $as }}>
