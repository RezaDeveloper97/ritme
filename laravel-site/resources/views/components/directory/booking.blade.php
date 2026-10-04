{{--
    <x-directory.booking :price-from="320000" unit="هر جلسه" phone="+982100000000"/>
    Booking panel of the place page (`#book`, the target of every «انتخاب» link). ┌ L5-04 SLOT ┐ The booking request
    form (service, preferred date, time window, name, phone) replaces the placeholder block marked
    `data-booking-slot` below; until then the panel shows the starting price and how to reach the place.
--}}
@props(['priceFrom' => null, 'unit' => null, 'phone' => null])
<aside id="book" aria-labelledby="book-title" {{ $attributes->class('box-content flex w-100 shrink-0 flex-col gap-4.5 rounded-6xl border border-line bg-surface p-7 shadow-booking max-sm:box-border max-sm:w-full max-sm:max-w-full') }}>
    <h2 id="book-title" class="sr-only">{{ __('directory.place.booking.label') }}</h2>
    @if ($priceFrom)
        <x-ui.price :amount="$priceFrom" from :unit="$unit ?? __('directory.place.booking.unit')" size="lg"/>
    @endif
    {{-- L5-04: booking form goes here (replace this block). --}}
    <div data-booking-slot class="flex flex-col gap-4.5">
        <p class="m-0 rounded-2xl bg-lavender px-4 py-3.5 text-[14.5px] leading-loose font-semibold text-ink">{{ __('directory.place.booking.soon') }}</p>
        @if ($phone)
            <x-ui.button :href="'tel:'.$phone" size="2xl" icon="phone" class="w-full justify-center">{{ __('directory.place.booking.call') }}</x-ui.button>
        @endif
        <x-ui.button href="#services" :variant="$phone ? 'outline' : 'primary'" size="2xl" class="w-full justify-center">{{ __('directory.place.booking.services') }}</x-ui.button>
        <x-ui.button href="#address" variant="outline" size="2xl" icon="navigate" class="w-full justify-center">{{ __('directory.place.booking.directions') }}</x-ui.button>
    </div>
    <p class="m-0 flex gap-2 text-sm leading-[1.8] font-semibold text-muted">
        <x-icon name="lock" class="mt-1 size-4 shrink-0 text-stage-teen"/>{{ __('directory.place.booking.privacy') }}
    </p>
</aside>
