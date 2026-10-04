{{-- Mock screen «cycle today» (AUDIT §2.4: index, cycle, ttc, teen): phase ring + fertile window + check-in. $data: stages/common.mock.cycle_today, or a stage's own `<stage>.mock.cycle_today` (teen: the school kit instead of the fertile window). --}}
<div class="flex flex-col items-center gap-3.5 px-4 py-6.5">
    <span class="text-xs font-bold text-on-night-muted">{{ $data['day'] ?? '' }}</span>
    <div class="flex size-45 items-center justify-center rounded-full bg-[conic-gradient(var(--color-stage-cycle)_0_17%,var(--color-lilac)_17%_45%,var(--color-phase-fertile)_45%_60%,var(--color-phase-luteal)_60%_100%)]">
        <div class="flex size-36.5 flex-col items-center justify-center gap-1 rounded-full bg-night">
            <span class="font-display text-[30px] text-on-night">{{ $data['countdown'] ?? '' }}</span>
            <span class="text-[11px] font-bold text-on-night-muted">{{ $data['countdown_label'] ?? '' }}</span>
        </div>
    </div>
    <div class="flex w-full justify-between rounded-2xl bg-night-card p-3 text-xs font-bold text-on-night">
        <span>{{ $data['fertile'] ?? '' }}</span>
        <span class="text-phase-fertile">{{ $data['fertile_value'] ?? '' }}</span>
    </div>
    <div class="flex w-full items-center gap-2 rounded-2xl bg-night-card p-3 text-xs font-bold text-on-night">
        <x-icon name="mic" class="size-4 text-lilac"/>{{ $data['checkin'] ?? '' }}
    </div>
</div>
