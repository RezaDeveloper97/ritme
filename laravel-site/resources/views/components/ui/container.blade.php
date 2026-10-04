{{--
    <x-ui.container as="div">…</x-ui.container> — page gutters: 120px inside the 1440 shell (1200 content), 20px ≤1024
    (AUDIT §3.4). Use x-ui.section for full-width bands with vertical padding.
--}}
@props(['as' => 'div'])
<{{ $as }} {{ $attributes->class('w-full px-30 max-lg:px-5') }}>{{ $slot }}</{{ $as }}>
