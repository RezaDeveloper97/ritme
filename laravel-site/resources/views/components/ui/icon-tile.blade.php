{{--
    <x-ui.icon-tile icon="drop" color="cycle" size="sm|md|lg" shape="circle|rounded"/>
    The tinted icon square/circle used by every card (AUDIT §2.3): a `color/10` background (`/18` for lilac on night)
    with the icon in the full colour. Colours: the six stages, primary, muted, lilac. Always decorative.
    sizes: sm 48 (icon 24, rounded-16) · md 56 (icon 28, rounded-18) · lg 120 medallion (icon 56).
--}}
@props(['icon', 'color' => 'primary', 'size' => 'md', 'shape' => 'rounded'])
@php
    $tint = match ($color) {
        'cycle' => 'bg-stage-cycle/10 text-stage-cycle',
        'ttc' => 'bg-stage-ttc/10 text-stage-ttc',
        'pregnancy' => 'bg-stage-pregnancy/10 text-stage-pregnancy',
        'postpartum' => 'bg-stage-postpartum/10 text-stage-postpartum',
        'menopause' => 'bg-stage-menopause/10 text-stage-menopause',
        'teen' => 'bg-stage-teen/10 text-stage-teen',
        'muted' => 'bg-muted/10 text-muted',
        'lilac' => 'bg-lilac/18 text-lilac',
        'primary-soft' => 'bg-primary/13 text-primary',
        'surface' => 'bg-surface text-stage-cycle shadow-card',
        default => 'bg-primary/10 text-primary',
    };
    [$box, $iconSize, $radius] = match ($size) {
        'sm' => ['size-12', 'size-6', 'rounded-xl'],
        'lg' => ['size-30', 'size-14', 'rounded-full'],
        default => ['size-14', 'size-7', 'rounded-2xl'],
    };
    $radius = $shape === 'circle' ? 'rounded-full' : $radius;
@endphp
<span aria-hidden="true" {{ $attributes->class("flex shrink-0 items-center justify-center {$box} {$radius} {$tint}") }}><x-icon :name="$icon" :class="$iconSize"/></span>
