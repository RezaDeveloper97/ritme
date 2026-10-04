{{--
    Mock screen «checklist» (feature splits): eyebrow, display title, progress bar, three ticked rows, «ادامه».
    $data: eyebrow, title, items (list), tint (period | fertile | luteal | follicular | stage key), cta.
--}}
@php
    [$solid, $soft, $text] = match ($data['tint'] ?? 'period') {
        'fertile' => ['bg-phase-fertile', 'bg-phase-fertile/20', 'text-phase-fertile'],
        'luteal' => ['bg-phase-luteal', 'bg-phase-luteal/20', 'text-phase-luteal'],
        'follicular' => ['bg-phase-follicular', 'bg-phase-follicular/20', 'text-phase-follicular'],
        'lilac' => ['bg-lilac', 'bg-lilac/20', 'text-lilac'],
        default => ['bg-phase-period', 'bg-phase-period/20', 'text-phase-period'],
    };
@endphp
<div class="flex flex-col gap-3 px-4 py-6.5">
    <span class="text-xs font-bold text-on-night-muted">{{ $data['eyebrow'] ?? '' }}</span>
    <span class="font-display text-[24px] leading-[1.4] text-on-night">{{ $data['title'] ?? '' }}</span>
    <div class="flex h-2 overflow-hidden rounded-full bg-night-card"><span class="w-[62%] {{ $solid }}"></span></div>
    @foreach (($data['items'] ?? []) as $item)
        <div class="flex items-center gap-2.5 rounded-xl bg-night-card p-3 text-start text-xs font-bold text-on-night">
            <span class="flex size-6.5 shrink-0 items-center justify-center rounded-full {{ $soft }}"><x-icon name="check" class="size-3.5 {{ $text }}"/></span>{{ $item }}
        </div>
    @endforeach
    <div class="mt-1 flex h-11 items-center justify-center rounded-full text-sm font-extrabold text-night {{ $solid }}">{{ $data['cta'] ?? '' }}</div>
</div>
