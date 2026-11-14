{{--
    <x-site.text-link href="#features">ببین چه دارد</x-site.text-link>
    Underlined 16/800 ink link with a trailing arrow (RTL: points left) — the redesign's secondary action.
--}}
@props(['href'])
<a href="{{ $href }}" {{ $attributes->class('inline-flex shrink-0 items-center gap-1.5 text-lg font-extrabold text-ink underline underline-offset-[6px] hover:text-primary') }}>{{ $slot }}<x-icon name="arrow-left" class="size-4"/></a>
