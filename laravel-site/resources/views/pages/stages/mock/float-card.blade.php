{{--
    Glass card floating next to the hero phone (AUDIT §2.4 `x-mock.float-card`): night-card/82, blur, shadow-float.
    $card: StageFloatCardData (icon, tint, title, text, position top-start | bottom-end). Decorative.
--}}
@php
    $tint = match ($card->tint) {
        'period' => 'bg-phase-period/15 text-phase-period',
        'fertile' => 'bg-phase-fertile/15 text-phase-fertile',
        'luteal' => 'bg-phase-luteal/15 text-phase-luteal',
        default => 'bg-lilac/15 text-lilac',
    };
@endphp
<div @class([
    'absolute flex min-w-52.5 items-center gap-3 rounded-[22px] border border-night-line bg-night-card/82 px-4 py-3.5 shadow-float backdrop-blur-sm',
    'top-22.5 -start-2.5' => $card->position === 'top-start',
    'bottom-27.5 -end-5' => $card->position !== 'top-start',
])>
    <span class="flex size-10.5 shrink-0 items-center justify-center rounded-lg {{ $tint }}"><x-icon :name="$card->icon" class="size-5.5"/></span>
    <span class="flex flex-col gap-0.5">
        <b class="text-base text-on-night">{{ $card->title }}</b>
        <span class="text-[12.5px] font-semibold text-on-night-muted">{{ $card->text }}</span>
    </span>
</div>
