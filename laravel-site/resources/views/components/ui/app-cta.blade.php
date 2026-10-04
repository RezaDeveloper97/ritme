{{--
    <x-ui.app-cta title="ریتمی را رایگان نصب کن" lead="…" :links="$appLinks" qr-url="https://…"/>
    <x-ui.app-cta variant="aside" icon="bell" title="یادآور و رزروهایت در اپ ریتمی" lead="…" :links="$appLinks"/>
    <x-ui.app-cta variant="strip" title="سفارش‌ها و لیست سیسمونی در اپ" lead="…" :links="$appLinks"/>
    The `#download` app CTA (AUDIT §2.4). `full`: night card, radius 40, p 64, page margins 0 120 96, h2 44 + lead +
    store badges + 180px QR tile (QR only when `qr-url` is given). `aside`: vertical night card for a sidebar
    (directory-booked); `strip`: a compact horizontal night band (shop-done). Heading/lead are per-page props;
    `links` = AppLinksSettings from the controller. Pass `id=""` to drop the #download anchor (only one per page).
--}}
@props([
    'title',
    'lead' => null,
    'links' => null,
    'qrUrl' => null,
    'variant' => 'full',
    'icon' => null,
    'id' => 'download',
    'headingId' => null,
])
@php
    $headingId ??= ($id !== '' ? $id : 'app-cta').'-title';
@endphp
@if ($variant === 'aside')
<section @if ($id !== '') id="{{ $id }}" @endif aria-labelledby="{{ $headingId }}" {{ $attributes->class('flex flex-col gap-4 rounded-6xl bg-night p-8 text-on-night') }}>
    @if ($icon)<x-icon :name="$icon" class="size-8.5 text-lilac" stroke="1.8"/>@endif
    <h2 id="{{ $headingId }}" class="m-0 font-display text-d-sm leading-heading font-normal text-on-night">{{ $title }}</h2>
    @if ($lead)<p class="m-0 text-md leading-loose font-medium text-on-night-muted">{{ $lead }}</p>@endif
    <x-ui.store-badges :links="$links"/>
    {{ $slot }}
</section>
@elseif ($variant === 'strip')
<section @if ($id !== '') id="{{ $id }}" @endif aria-labelledby="{{ $headingId }}" {{ $attributes->class('flex items-center gap-7 rounded-6xl bg-night px-10 py-8 text-start max-lg:flex-wrap max-sm:px-6') }}>
    <div class="flex flex-[1_1_15rem] flex-col gap-2 max-sm:basis-full">
        <h2 id="{{ $headingId }}" class="m-0 font-display text-d-sm leading-heading font-normal text-on-night">{{ $title }}</h2>
        @if ($lead)<p class="m-0 text-md leading-loose font-medium text-on-night-muted">{{ $lead }}</p>@endif
    </div>
    <x-ui.store-badges :links="$links"/>
</section>
@else
<section @if ($id !== '') id="{{ $id }}" @endif aria-labelledby="{{ $headingId }}" {{ $attributes->class('mx-30 mt-0 mb-24 flex items-center gap-12 overflow-hidden rounded-7xl bg-night p-16 max-lg:flex-wrap max-sm:mx-5 max-sm:mb-12 max-sm:rounded-4xl') }}>
    <div class="flex flex-1 flex-col gap-4.5 max-lg:basis-75 max-sm:basis-full">
        <h2 id="{{ $headingId }}" class="m-0 font-display text-d-xl leading-heading font-normal text-on-night">{{ $title }}</h2>
        @if ($lead)<p class="m-0 text-xl leading-loose font-medium text-on-night-muted">{{ $lead }}</p>@endif
        <x-ui.store-badges :links="$links"/>
        {{ $slot }}
    </div>
    <x-ui.qr :url="$qrUrl"/>
</section>
@endif
