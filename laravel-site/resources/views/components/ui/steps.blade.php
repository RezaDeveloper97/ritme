{{--
    <x-ui.steps :items="[['title' => 'معرفی مجموعه', 'text' => 'نام، نوع خدمت، شهر و راه تماس'], …]"/>
    Numbered steps in a row (AUDIT §2.2, directory-business «چطور ثبت کنم؟»): 56px primary circle with a Persian digit
    (Lalezar 26) + title 19 + text 15. An ordered list; wraps to two columns ≤1024 and one ≤700.
    `tone="dark"` (index «یک اپ، پنج بخش» band) uses lilac numbers on night.
--}}
@props(['items' => [], 'tone' => 'light'])
@php($dark = $tone === 'dark')
<ol {{ $attributes->class('m-0 flex list-none gap-8 p-0 max-lg:flex-wrap') }}>
    @foreach ($items as $item)
        <li class="flex flex-1 flex-col gap-3 max-lg:basis-75 max-sm:basis-full">
            <span aria-hidden="true" @class(['flex size-14 items-center justify-center rounded-full font-display text-[26px]', $dark ? 'bg-lilac text-night' : 'bg-primary text-white'])>{{ \App\View\Components\Layout\Footer::persianDigits((string) $loop->iteration) }}</span>
            <b @class(['text-[19px]', $dark ? 'text-on-night' : 'text-ink'])><span class="sr-only">گام {{ \App\View\Components\Layout\Footer::persianDigits((string) $loop->iteration) }}: </span>{{ $item['title'] }}</b>
            @if (! empty($item['text']))<span @class(['text-md leading-relaxed font-medium', $dark ? 'text-on-night-muted' : 'text-muted'])>{{ $item['text'] }}</span>@endif
        </li>
    @endforeach
</ol>
