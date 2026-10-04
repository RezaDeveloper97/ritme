{{--
    Mock screen «همدم» (AUDIT §2.4: shared by home, ttc, postpartum, teen): the companion's view of a shared cycle —
    greeting, cycle card with a phase bar, today's tip, one reminder row. Decorative.
    $data: eyebrow, title, card_label, card_value, tip, row_label, row_value — a stage overrides it with
    `<stage>.mock.companion` (teen: the mother's view), otherwise stages/common.mock.companion (the design's copy).
--}}
@php
    $data = ($data ?? []) !== [] ? $data : __('stages/common.mock.companion');
@endphp
<div class="flex flex-col gap-3 px-4 py-6.5">
    <span class="text-xs font-bold text-on-night-muted">{{ $data['eyebrow'] ?? '' }}</span>
    <span class="font-display text-d-sm text-on-night">{{ $data['title'] ?? '' }}</span>
    <div class="flex flex-col gap-2 rounded-2xl bg-night-card p-3.5">
        <div class="text-[11px] font-bold text-on-night-muted">{{ $data['card_label'] ?? '' }}</div>
        <b class="font-display text-4xl text-on-night">{{ $data['card_value'] ?? '' }}</b>
        <div class="h-2 rounded-full bg-[linear-gradient(to_left,var(--color-phase-period)_0_17%,var(--color-lilac)_17%_45%,var(--color-phase-fertile)_45%_60%,var(--color-phase-luteal)_60%)]"></div>
    </div>
    <div class="rounded-2xl bg-night-card p-3 text-xs font-bold text-on-night">{{ $data['tip'] ?? '' }}</div>
    <div class="flex justify-between rounded-2xl bg-night-card p-3 text-xs font-bold text-on-night">
        <span>{{ $data['row_label'] ?? '' }}</span>
        <span class="text-phase-fertile">{{ $data['row_value'] ?? '' }}</span>
    </div>
</div>
