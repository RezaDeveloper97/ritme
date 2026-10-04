{{--
    Mock screen «baby today» (postpartum hero + first split): child age, weight growth chart, three status rows.
    Copy: stages/postpartum.mock.baby (the shared builder only passes stages/common.mock.* to non-checklist screens).
--}}
@php
    $copy = __('stages/postpartum.mock.baby');
    $tints = ['luteal' => 'text-phase-luteal', 'fertile' => 'text-phase-fertile', 'lilac' => 'text-lilac'];
@endphp
<div class="flex flex-col gap-3 px-4 py-6.5">
    <span class="text-xs font-bold text-on-night-muted">{{ $copy['label'] ?? '' }}</span>
    <div class="rounded-2xl bg-night-card p-3.5">
        <div class="text-[11px] font-bold text-on-night-muted">{{ $copy['chart'] ?? '' }}</div>
        <svg width="100%" height="90" viewBox="0 0 200 90" preserveAspectRatio="none" aria-hidden="true" class="inline align-baseline">
            <path d="M0,80 C60,60 120,35 200,20" fill="none" stroke-width="14" class="stroke-night-line"/>
            <polyline points="0,78 40,66 80,52 120,40 160,30" fill="none" stroke-width="3" class="stroke-phase-fertile"/>
        </svg>
    </div>
    @foreach (($copy['rows'] ?? []) as $row)
        <div class="flex justify-between rounded-lg bg-night-card p-3 text-xs font-bold text-on-night">
            <span>{{ $row['label'] }}</span>
            <span class="{{ $tints[$row['tint']] ?? 'text-lilac' }}">{{ $row['value'] }}</span>
        </div>
    @endforeach
</div>
