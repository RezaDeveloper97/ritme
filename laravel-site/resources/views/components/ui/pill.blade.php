{{--
    <x-ui.pill tone="teen|primary|surface|danger" dot>امروز ۱۷:۰۰</x-ui.pill>
    <x-ui.pill tone="surface" icon="shield-check" icon-class="text-stage-teen">مدارک بررسی شد</x-ui.pill>
    Rounder sibling of x-ui.badge for the directory/shop facts (AUDIT §2.2): next-slot chips with a status dot
    (26px, 11.5/700, `teen/12`), the verified badge on covers, open-now, product corner labels («پنبه ۱۰۰٪»). Use
    x-ui.badge for plain status labels; this one adds the dot and the cover-overlay sizes. size: xs 22 · sm 26 · md 30.
--}}
@props(['tone' => 'teen', 'dot' => false, 'icon' => null, 'iconClass' => 'text-primary', 'size' => 'sm'])
@php
    [$toneClasses, $dotClass] = match ($tone) {
        'primary' => ['bg-primary/13 text-ink', 'bg-primary'],
        'surface' => ['bg-surface text-ink', 'bg-stage-teen'],
        'danger' => ['bg-danger-soft text-danger', 'bg-danger'],
        'muted' => ['border border-line bg-canvas text-muted', 'bg-muted'],
        default => ['bg-stage-teen/12 text-ink', 'bg-stage-teen'],
    };
    $sizeClasses = match ($size) {
        'xs' => 'h-5.5 px-2 text-2xs font-extrabold',
        'md' => 'h-7.5 px-3 text-xs font-extrabold',
        default => 'h-6.5 px-2.5 text-[11.5px] font-bold',
    };
@endphp
<span {{ $attributes->class("inline-flex items-center gap-[5px] rounded-full whitespace-nowrap {$sizeClasses} {$toneClasses}") }}>@if ($dot)<span aria-hidden="true" class="size-[7px] rounded-full {{ $dotClass }}"></span>@endif @if ($icon)<x-icon :name="$icon" class="size-[15px] {{ $iconClass }}"/>@endif{{ $slot }}</span>
