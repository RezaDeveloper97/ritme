{{--
    <x-ui.success-hero icon="check" tone="success|lavender" title="سفارشت ثبت شد">کد سفارش ۵۲۰۸۴۱ · …</x-ui.success-hero>
    Confirmation header (AUDIT §2.2: directory-booked, directory-join-done, shop-done): 96–104px icon circle with a soft
    ring + the page's only h1 (`as` to demote it) (Lalezar 52) + the lead (slot) carrying the tracking/order code. Centered by default;
    `align="start"` for the booked page's two-column layout. The `actions` slot renders a button row under the lead.
--}}
@props(['icon' => 'check', 'tone' => 'success', 'title', 'align' => 'center', 'as' => 'h1'])
@php
    $circle = $tone === 'lavender'
        ? 'size-26 bg-lavender ring-16 ring-lavender/53 text-primary'
        : 'size-24 bg-stage-teen/12 ring-14 ring-stage-teen/5 text-stage-teen';
    $center = $align === 'center';
@endphp
<div {{ $attributes->class(['flex flex-col gap-5.5', 'items-center text-center' => $center, 'items-start' => ! $center]) }}>
    <span aria-hidden="true" class="flex shrink-0 items-center justify-center rounded-full {{ $circle }}"><x-icon :name="$icon" class="size-11.5"/></span>
    <{{ $as }} class="m-0 font-display text-[52px] leading-display font-normal text-ink max-lg:text-[38px] max-sm:text-[30px]">{{ $title }}</{{ $as }}>
    @if (trim((string) $slot) !== '')
        <p @class(['m-0 max-w-160 text-xl leading-loose font-medium text-muted max-sm:max-w-full', 'text-center' => $center])>{{ $slot }}</p>
    @endif
    @isset($actions)
        <div class="flex flex-wrap gap-3">{{ $actions }}</div>
    @endisset
</div>
