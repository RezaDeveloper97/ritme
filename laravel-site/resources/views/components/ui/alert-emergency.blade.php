{{--
    <x-ui.alert-emergency number="115">اگر خونریزی خیلی زیاد داری… با پزشک یا اورژانس تماس بگیر.</x-ui.alert-emergency>
    Emergency note (AUDIT §2.2): danger-soft fill, danger-line border, radius 24, white circle with alert-triangle, the
    text, and the «۱۱۵» pill as a real `tel:` link. `role="note"` (it is advice, not a live alert). The number comes
    from settings (`general.emergency_number`) via the page; default 115.
--}}
@props(['number' => '115'])
@php($numberLabel = \App\View\Components\Layout\Footer::persianDigits((string) $number))
<div role="note" {{ $attributes->class('flex items-center gap-4 rounded-4xl border border-danger-line bg-danger-soft px-6.5 py-5.5 max-lg:flex-wrap') }}>
    <span aria-hidden="true" class="flex size-12 shrink-0 items-center justify-center rounded-full bg-surface"><x-icon name="alert-triangle" class="size-6 text-danger" stroke="1.8"/></span>
    <p class="m-0 grow text-lg leading-relaxed font-semibold text-ink">{{ $slot }}</p>
    <a href="tel:{{ $number }}" aria-label="تماس با اورژانس {{ $numberLabel }}" class="flex h-12 items-center rounded-full bg-danger px-5.5 text-xl font-extrabold text-white transition-colors hover:opacity-90 hover:text-white">{{ $numberLabel }}</a>
</div>
