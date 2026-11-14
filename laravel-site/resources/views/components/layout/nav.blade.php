{{--
    <x-layout.nav :links="list<NavLinkData>" :variant="HeaderVariant"/> — desktop main nav (hidden ≤1024, where the
    mobile menu takes over). Nov 2026 redesign: 40px pills; the active item is weight 800, ink, on a canvas pill.
    aria-current="page" on the link to the current page, "true" on the section item.
--}}
@props(['links', 'variant'])
@php($dark = $variant->isDark())
<nav aria-label="منوی اصلی" class="flex gap-1 max-lg:hidden">
    @foreach ($links as $link)
        <a href="{{ $link->url }}" @if ($link->ariaCurrent() !== null) aria-current="{{ $link->ariaCurrent() }}" @endif @class([
            'flex h-10 items-center rounded-full px-4 text-md max-xl:px-3 max-xl:text-base',
            'font-extrabold' => $link->active,
            'font-semibold' => ! $link->active,
            'bg-canvas text-ink hover:text-ink' => ! $dark && $link->active,
            'text-muted hover:bg-canvas hover:text-ink' => ! $dark && ! $link->active,
            'bg-night-card text-on-night hover:text-on-night' => $dark && $link->active,
            'text-on-night-muted hover:text-on-night' => $dark && ! $link->active,
        ])>{{ $link->label }}</a>
    @endforeach
</nav>
