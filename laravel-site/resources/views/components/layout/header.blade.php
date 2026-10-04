{{--
    Header markup, rendered (and fragment-cached) by App\View\Components\Layout\Header — use <x-layout.header/>.
    Data: $variant (HeaderVariant), $links (list<NavLinkData>), $homeUrl, $siteName, $loginUrl (?string),
    $downloadHref. Grid placement (row 1, menu row 2) matches layouts/app.
--}}
@php($dark = $variant->isDark())
@php($tone = $dark ? 'dark' : 'light')
<header @class([
    'relative col-start-1 row-start-1 flex items-center justify-between px-30 py-5 max-xl:px-6 max-xl:py-3.5 max-lg:p-5',
    'border-b border-line bg-surface' => ! $dark,
])>
    <a href="{{ $homeUrl }}" @class(['flex items-center gap-2.5', 'text-on-night hover:text-on-night' => $dark, 'text-ink hover:text-ink' => ! $dark])>
        <span @class(['flex size-10 items-center justify-center rounded-full', 'bg-lilac' => $dark, 'bg-primary' => ! $dark])><x-icon name="drop" class="size-5 text-white"/></span>
        <span class="font-display text-[30px] leading-none">{{ $siteName }}</span>
    </a>
    <x-layout.nav :links="$links" :variant="$variant"/>
    <div class="flex items-center gap-2.5 max-lg:hidden">
        @if ($loginUrl !== null)
            <x-ui.button :href="$loginUrl" variant="outline" :tone="$tone" icon="user" :icon-class="$dark ? 'size-4.5 text-lilac' : 'size-4.5 text-primary'">ورود</x-ui.button>
        @endif
        <x-ui.button :href="$downloadHref" :tone="$tone" icon="download" icon-class="size-[17px]">دانلود اپ</x-ui.button>
    </div>
    <button type="button" data-module="menu" aria-controls="site-menu" aria-expanded="false" aria-label="باز کردن منو" data-label-close="بستن منو" @class([
        'group ms-3 hidden size-11.5 flex-col items-center justify-center gap-[5px] rounded-full border-[1.5px] bg-transparent max-lg:flex',
        'border-night-line text-on-night' => $dark,
        'border-line text-ink' => ! $dark,
    ])>
        <span class="block h-0.5 w-5 rounded-full bg-current transition-transform motion-reduce:transition-none group-aria-expanded:translate-y-[7px] group-aria-expanded:rotate-45"></span>
        <span class="block h-0.5 w-5 rounded-full bg-current group-aria-expanded:opacity-0"></span>
        <span class="block h-0.5 w-5 rounded-full bg-current transition-transform motion-reduce:transition-none group-aria-expanded:-translate-y-[7px] group-aria-expanded:-rotate-45"></span>
    </button>
</header>
<x-layout.mobile-menu :links="$links" :variant="$variant" :login-url="$loginUrl" :download-href="$downloadHref"/>
