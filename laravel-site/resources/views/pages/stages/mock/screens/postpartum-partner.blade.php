{{--
    Mock screen «partner» (postpartum family split; teen-mother reuses it): greeting, shared cycle card with a phase
    bar, today's tip, one reminder row. Copy: $copy, default stages/postpartum.mock.partner.
--}}
@php
    $copy = $copy ?? __('stages/postpartum.mock.partner');
@endphp
<div class="flex flex-col gap-3 px-4 py-6.5">
    <span class="text-xs font-bold text-on-night-muted">{{ $copy['eyebrow'] ?? '' }}</span>
    <span class="font-display text-d-sm text-on-night">{{ $copy['title'] ?? '' }}</span>
    <div class="flex flex-col gap-2 rounded-2xl bg-night-card p-3.5">
        <div class="text-[11px] font-bold text-on-night-muted">{{ $copy['card_label'] ?? '' }}</div>
        <b class="font-display text-4xl text-on-night">{{ $copy['card_value'] ?? '' }}</b>
        <div class="h-2 rounded-full bg-[linear-gradient(to_left,var(--color-phase-period)_0_17%,var(--color-lilac)_17%_45%,var(--color-phase-fertile)_45%_60%,var(--color-phase-luteal)_60%)]"></div>
    </div>
    <div class="rounded-2xl bg-night-card p-3 text-xs font-bold text-on-night">{{ $copy['tip'] ?? '' }}</div>
    <div class="flex justify-between rounded-2xl bg-night-card p-3 text-xs font-bold text-on-night">
        <span>{{ $copy['row_label'] ?? '' }}</span>
        <span class="text-phase-fertile">{{ $copy['row_value'] ?? '' }}</span>
    </div>
</div>
