{{--
    <x-layout.nav :links="list<NavLinkData>" :variant="HeaderVariant"/> — desktop main nav (hidden ≤1024, where the
    mobile menu takes over). Active item: weight 800, ink colour, 2px underline; aria-current="page" on the link
    to the current page, "true" on the section item. The «مرحله‌ها» chevron is decorative (no dropdown).
--}}
@props(['links', 'variant'])
@php($dark = $variant->isDark())
<nav aria-label="منوی اصلی" class="flex gap-6.5 max-xl:gap-3.5 max-lg:hidden">
    @foreach ($links as $link)
        <a href="{{ $link->url }}" @if ($link->ariaCurrent() !== null) aria-current="{{ $link->ariaCurrent() }}" @endif @class([
            'flex items-center gap-1 border-b-2 px-0.5 py-2.5 text-md max-xl:text-base',
            'font-extrabold' => $link->active,
            'font-semibold border-b-transparent' => ! $link->active,
            'border-b-lilac text-on-night hover:text-on-night' => $dark && $link->active,
            'text-on-night-muted hover:text-on-night' => $dark && ! $link->active,
            'border-b-primary text-ink hover:text-ink' => ! $dark && $link->active,
            'text-muted hover:text-ink' => ! $dark && ! $link->active,
        ])>{{ $link->label }}@if ($link->chevron)<x-icon name="chevron-left" @class(['size-[13px]', 'text-on-night-muted' => $dark, 'text-muted' => ! $dark])/>@endif</a>
    @endforeach
</nav>
