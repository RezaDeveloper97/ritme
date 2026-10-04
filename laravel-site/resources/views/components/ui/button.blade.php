{{--
    <x-ui.button href="…" variant="primary|outline|ghost|danger" tone="light|dark" size="sm|md|lg|xl|2xl"
                 icon="download" icon-class="size-4.5 text-lilac" icon-end="arrow-left">متن</x-ui.button>
    AUDIT §2.2: fully rounded pills; heights 40 / 46 / 52 / 54 / 56 (outline: + its border, content-box as in the design). `tone="dark"` is for night backgrounds.
    Renders <a> when href is given, otherwise <button type="button"> (pass type="submit" for forms).
--}}
@props([
    'href' => null,
    'variant' => 'primary',
    'tone' => 'light',
    'size' => 'md',
    'icon' => null,
    'iconClass' => 'size-4.5',
    'iconEnd' => null,
])
@php
    $dark = $tone === 'dark';
    $ghost = $variant === 'ghost';
    $sizeClasses = $ghost ? 'text-md' : match ($size) {
        'sm' => 'h-10 text-base',
        'lg' => 'h-13 text-md',
        'xl' => 'h-13.5 text-md',
        '2xl' => 'h-14 text-lg',
        default => 'h-11.5 text-base',
    };
    $padding = $ghost ? '' : match ($size) {
        'sm' => 'px-4',
        'lg' => 'px-6',
        'xl' => 'px-6.5',
        '2xl' => 'px-7',
        default => $variant === 'outline' ? 'px-4.5' : 'px-5',
    };
    $variantClasses = match ($variant) {
        'outline' => $dark
            ? 'box-content border-[1.5px] border-night-line text-on-night hover:border-lilac hover:text-on-night'
            : 'box-content border-[1.5px] border-line text-ink hover:border-primary hover:text-ink',
        'ghost' => $dark ? 'text-lilac hover:text-on-night' : 'text-primary hover:text-primary-hover',
        'danger' => 'bg-danger text-white hover:text-white',
        default => $dark
            ? 'bg-lilac text-night hover:bg-on-night hover:text-night'
            : 'bg-primary text-white hover:bg-primary-hover hover:text-white',
    };
    $classes = "inline-flex items-center gap-2 rounded-full font-extrabold whitespace-nowrap transition-colors {$sizeClasses} {$padding} {$variantClasses}";
@endphp
@if ($href !== null)
<a href="{{ $href }}" {{ $attributes->class($classes) }}>@if ($icon)<x-icon :name="$icon" :class="$iconClass"/>@endif{{ $slot }}@if ($iconEnd)<x-icon :name="$iconEnd" :class="$iconClass"/>@endif</a>
@else
<button {{ $attributes->merge(['type' => 'button'])->class($classes) }}>@if ($icon)<x-icon :name="$icon" :class="$iconClass"/>@endif{{ $slot }}@if ($iconEnd)<x-icon :name="$iconEnd" :class="$iconClass"/>@endif</button>
@endif
