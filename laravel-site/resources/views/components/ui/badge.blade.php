{{--
    <x-ui.badge tone="lavender|success|danger|warn|info|night|surface" icon="shield-check">مدارک بررسی شد</x-ui.badge>
    Small status pill (AUDIT §2.2). Stage-coloured labels: pass a class (`bg-stage-ttc/10 text-stage-ttc`) with tone="none".
--}}
@props(['tone' => 'lavender', 'icon' => null])
@php
    $toneClasses = match ($tone) {
        'success' => 'bg-success-soft text-success',
        'danger' => 'bg-danger-soft text-danger',
        'warn' => 'bg-warn-bg text-warn-text',
        'info' => 'bg-info-soft text-stage-postpartum',
        'night' => 'bg-night/80 text-on-night',
        'surface' => 'bg-surface text-ink',
        'none' => '',
        default => 'bg-lavender text-primary',
    };
@endphp
<span {{ $attributes->class("inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs leading-none font-extrabold whitespace-nowrap {$toneClasses}") }}>@if ($icon)<x-icon :name="$icon" class="size-3.5"/>@endif{{ $slot }}</span>
