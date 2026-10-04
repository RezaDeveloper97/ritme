{{--
    <x-layout.mobile-menu :links :variant :login-url :download-href/> — the ≤1024 menu (#site-menu), toggled by the
    header burger through the `menu` data-module (aria-expanded, `hidden`, Esc closes and returns focus). Closed and
    never shown above 1024px. Without JS it stays closed; the footer keeps every link reachable.
--}}
@props(['links', 'variant', 'loginUrl' => null, 'downloadHref'])
@php($dark = $variant->isDark())
@php($tone = $dark ? 'dark' : 'light')
<div id="site-menu" hidden @class([
    'relative col-start-1 row-start-2 border-b px-5 pt-3 pb-5 lg:hidden',
    'border-night-line bg-night text-on-night' => $dark,
    'border-line bg-surface text-ink' => ! $dark,
])>
    <nav aria-label="منوی موبایل">
        <ul class="flex flex-col">
            @foreach ($links as $link)
                <li><a href="{{ $link->url }}" @if ($link->ariaCurrent() !== null) aria-current="{{ $link->ariaCurrent() }}" @endif @class([
                    'flex min-h-12 items-center border-b border-primary/12 text-lg font-bold',
                    'text-on-night hover:text-lilac aria-[current]:text-lilac' => $dark,
                    'text-ink hover:text-primary aria-[current]:text-primary' => ! $dark,
                ])>{{ $link->label }}</a></li>
            @endforeach
        </ul>
    </nav>
    <div class="flex flex-wrap gap-2.5 pt-3.5">
        @if ($loginUrl !== null)
            <x-ui.button :href="$loginUrl" variant="outline" :tone="$tone" icon="user" :icon-class="$dark ? 'size-4.5 text-lilac' : 'size-4.5 text-primary'" class="flex-[1_1_140px] justify-center">ورود</x-ui.button>
        @endif
        <x-ui.button :href="$downloadHref" :tone="$tone" icon="download" icon-class="size-[17px]" class="flex-[1_1_140px] justify-center">دانلود اپ</x-ui.button>
    </div>
</div>
