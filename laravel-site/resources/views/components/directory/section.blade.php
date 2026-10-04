{{--
    <x-directory.section id="services" :title="__('…')" :lead="__('…')">…</x-directory.section>
    A block of the place page (L5-03): top hairline, 32px vertical padding, h2 22/800 + optional muted lead, labelled by
    its heading. `id` is also the anchor (#services, #reviews, #address …).
--}}
@props(['id', 'title', 'lead' => null])
<section id="{{ $id }}" aria-labelledby="{{ $id }}-title" {{ $attributes->class('flex flex-col gap-4.5 border-t border-t-line py-8') }}>
    <div>
        <h2 id="{{ $id }}-title" class="m-0 font-sans text-4xl leading-normal font-extrabold">{{ $title }}</h2>
        @if ($lead)<p class="mt-1 mb-0 text-sm-plus font-semibold text-muted">{{ $lead }}</p>@endif
    </div>
    {{ $slot }}
</section>
