{{--
    <x-ui.price :amount="485000" :compare="460000"/>             →  ۴۸۵ هزار تومان  ~~۴۶۰ هزار~~
    <x-ui.price :amount="320000" from unit="هر جلسه" size="lg"/>  →  از ۳۲۰ هزار تومان · هر جلسه (Lalezar 30)
    <x-ui.price :amount="320000" from size="inline"/>             →  inside running text (place card)
    Amounts are integer tomans, formatted by App\Support\Text\Toman: whole thousands read «… هزار», others are
    grouped with «٬»; digits are Persian (AUDIT §2.2). `compare` (compare-at / old price) is struck through with an
    sr-only «قیمت قبلی». No currency maths here — controllers pass ready integers (Money value object: L6-01).
--}}
@props(['amount', 'compare' => null, 'from' => false, 'unit' => null, 'size' => 'md', 'currency' => 'تومان'])
@php
    $amountText = \App\Support\Text\Toman::format((int) $amount);
    $compareText = $compare !== null ? \App\Support\Text\Toman::format((int) $compare) : null;
@endphp
@if ($size === 'inline')
<span {{ $attributes }}>@if ($from)از @endif<b class="text-ink">{{ $amountText }}</b> {{ $currency }}@if ($unit) · {{ $unit }}@endif</span>
@elseif ($size === 'lg')
<span {{ $attributes->class('inline-flex flex-wrap items-baseline gap-1.5') }}>
    @if ($from)<span class="text-sm font-bold text-muted">از</span>@endif
    <b class="font-display text-[30px] font-normal">{{ $amountText }}</b>
    <span class="text-base font-semibold text-muted">{{ $currency }}@if ($unit) · {{ $unit }}@endif</span>
    @if ($compareText)<s class="text-base font-semibold text-muted"><span class="sr-only">قیمت قبلی: </span>{{ $compareText }}</s>@endif
</span>
@else
<span {{ $attributes->class('inline-flex flex-wrap items-baseline gap-1.5') }}>
    @if ($from)<span class="text-[12.5px] font-bold text-muted">از</span>@endif
    <b class="text-lg">{{ $amountText }} <span class="text-[12.5px] font-bold text-muted">{{ $currency }}@if ($unit) · {{ $unit }}@endif</span></b>
    @if ($compareText)<s class="text-[12.5px] font-semibold text-muted"><span class="sr-only">قیمت قبلی: </span>{{ $compareText }}</s>@endif
</span>
@endif
