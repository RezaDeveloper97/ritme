{{--
    <x-ui.rating :value="4.8" :count="126" size="sm|md"/>   →  ★ ۴٫۸ (۱۲۶)
    <x-ui.rating :value="4" stars/>                          →  five-star row (reviews only)
    Star + score with Persian digits and «٫» decimal (AUDIT §2.2). The visual is aria-hidden; screen readers get
    «امتیاز ۴٫۸ از ۵ از ۱۲۶ نظر». size: sm 12.5/13 (cards, reviews) · md 13.5 (place card).
--}}
@props(['value', 'count' => null, 'size' => 'sm', 'stars' => false])
@php
    $fa = static fn (string $v): string => \App\View\Components\Layout\Footer::persianDigits(str_replace('.', '٫', $v));
    $score = (float) $value;
    $scoreText = $fa(rtrim(rtrim(number_format($score, 1, '.', ''), '0'), '.'));
    $countText = $count !== null ? $fa(number_format((int) $count, 0, '.', '٬')) : null;
    $spoken = "امتیاز {$scoreText} از ۵".($countText !== null ? " از {$countText} نظر" : '');
    $textSize = match ($size) {
        'md' => 'text-sm-plus',
        'xs' => 'text-[12.5px]',
        default => 'text-sm',
    };
@endphp
<span {{ $attributes->class("inline-flex items-center gap-1 font-extrabold {$textSize}") }}>
    <span class="sr-only">{{ $spoken }}</span>
    @if ($stars)
        <span aria-hidden="true" class="inline-flex gap-0.5">@for ($i = 1; $i <= 5; $i++)<x-icon name="star" @class(['size-3.5', 'fill-stage-ttc text-stage-ttc' => $i <= round($score), 'text-line' => $i > round($score)])/>@endfor</span>
    @else
        <x-icon name="star" class="size-3.5 text-stage-ttc"/>
    @endif
    <span aria-hidden="true">{{ $scoreText }}@if ($countText !== null)<span class="ms-1 font-semibold text-muted">({{ $countText }})</span>@endif</span>
</span>
