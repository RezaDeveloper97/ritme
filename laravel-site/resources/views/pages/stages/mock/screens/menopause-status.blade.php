{{--
    Mock screen «menopause status» (menopause hero + first split): months without a period, three symptom counters,
    symptom score with a trend line, «hot flash now» button. Copy: stages/menopause.mock.status.
--}}
@php
    $copy = __('stages/menopause.mock.status');
    $tints = ['period' => 'text-phase-period', 'lilac' => 'text-lilac', 'fertile' => 'text-phase-fertile'];
@endphp
<div class="flex flex-col gap-3 px-4 py-6.5">
    <span class="text-xs font-bold text-on-night-muted">{{ $copy['label'] ?? '' }}</span>
    <span class="font-display text-d-sm text-on-night">{{ $copy['title'] ?? '' }}</span>
    <div class="grid grid-cols-3 gap-1.5 max-sm:grid-cols-1">
        @foreach (($copy['stats'] ?? []) as $stat)
            <div class="rounded-lg bg-night-card px-1.5 py-2.5 text-center">
                <b class="block font-display text-3xl {{ $tints[$stat['tint']] ?? 'text-lilac' }}">{{ $stat['value'] }}</b>
                <span class="text-[10px] font-bold text-on-night-muted">{{ $stat['label'] }}</span>
            </div>
        @endforeach
    </div>
    <div class="rounded-2xl bg-night-card p-3.5">
        <div class="text-[11px] font-bold text-on-night-muted">{{ $copy['score_label'] ?? '' }}</div>
        <b class="font-display text-[26px] text-on-night">{{ $copy['score'] ?? '' }}</b>
        <svg width="100%" height="50" viewBox="0 0 200 50" preserveAspectRatio="none" aria-hidden="true" class="inline align-baseline">
            <polyline points="200,20 160,12 120,6 80,10 40,4 0,6" fill="none" stroke-width="3" class="stroke-lilac"/>
        </svg>
    </div>
    <div class="flex items-center gap-2 rounded-2xl border border-phase-period bg-phase-period/13 p-3 text-xs font-bold text-on-night">
        <x-icon name="flame" class="size-4 text-phase-period"/>{{ $copy['action'] ?? '' }}
    </div>
</div>
