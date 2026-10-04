{{--
    Mock screen «pregnancy week 24» (AUDIT §2.4, /pregnancy hero + «هفته‌به‌هفته» split): glowing egg medallion,
    size comparison, 2×2 stat tiles. $data: stages/pregnancy.mock.pregnancy_week. Decorative.
--}}
@php
    $week = $data ?? [];
@endphp
<div class="flex flex-col items-center gap-3.5 px-4 py-6.5">
    <span class="text-xs font-bold text-on-night-muted">{{ $week['label'] ?? '' }}</span>
    <div class="flex size-42.5 items-center justify-center rounded-full bg-[radial-gradient(circle,color-mix(in_srgb,var(--color-phase-luteal)_33%,transparent),color-mix(in_srgb,var(--color-phase-luteal)_6%,transparent)_70%)]">
        <x-icon name="egg" class="size-17.5 text-phase-luteal"/>
    </div>
    <span class="font-display text-[24px] text-on-night">{{ $week['size'] ?? '' }}</span>
    <div class="grid w-full grid-cols-2 gap-2 max-sm:grid-cols-1">
        @foreach (($week['stats'] ?? []) as $stat)
            <div class="rounded-lg bg-night-card p-2.5 text-[11px] font-bold text-on-night-muted"><b class="block text-md text-on-night">{{ $stat['value'] ?? '' }}</b>{{ $stat['label'] ?? '' }}</div>
        @endforeach
    </div>
</div>
