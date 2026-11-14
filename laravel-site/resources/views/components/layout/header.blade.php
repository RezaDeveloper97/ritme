{{--
    Header markup, rendered (and fragment-cached) by App\View\Components\Layout\Header — use <x-layout.header/>.
    Data: $variant (HeaderVariant), $links (list<NavLinkData>), $homeUrl, $siteName, $loginUrl (?string),
    $downloadHref, $logoUrl. Grid placement (row 1, menu row 2) matches layouts/app.
    Nov 2026 redesign: 80px bar, logo (the app's «R» ring) · centred pill nav · ink «دانلود اپ». The dark variant keeps the
    same shape in night colours for the pages whose hero is still dark.
--}}
@php($dark = $variant->isDark())
<header @class([
    'bleed col-start-1 row-start-1 grid h-20 grid-cols-[1fr_auto_1fr] items-center px-30 max-xl:px-6 max-lg:flex max-lg:h-16 max-lg:justify-between max-lg:px-5',
    'border-b border-line bg-surface' => ! $dark,
    'border-b border-night-line' => $dark,
])>
    <a href="{{ $homeUrl }}" @class(['flex items-center gap-2 justify-self-start', 'text-on-night hover:text-on-night' => $dark, 'text-ink hover:text-ink' => ! $dark])>
        <img src="{{ $logoUrl }}" width="106" height="40" alt="{{ $siteName }}" class="h-10 w-auto shrink-0" decoding="async">
    </a>
    <x-layout.nav :links="$links" :variant="$variant"/>
    <div class="flex items-center gap-2.5 justify-self-end max-lg:hidden">
        @if ($loginUrl !== null)
            <a href="{{ $loginUrl }}" @class([
                'flex h-11 items-center gap-2 rounded-full border px-5 text-[14.5px] font-extrabold',
                'border-line text-ink hover:bg-canvas hover:text-ink' => ! $dark,
                'border-night-line text-on-night hover:text-on-night' => $dark,
            ])><x-icon name="user" class="size-4"/>ورود</a>
        @endif
        <a href="{{ $downloadHref }}" @class([
            'flex h-11 items-center gap-2 rounded-full px-5.5 text-[14.5px] font-extrabold',
            'bg-ink text-white hover:bg-ink/90 hover:text-white' => ! $dark,
            'bg-surface text-ink hover:text-ink' => $dark,
        ])><x-icon name="download" class="size-[17px]"/>دانلود اپ</a>
    </div>
    <button type="button" data-module="menu" aria-controls="site-menu" aria-expanded="false" aria-label="باز کردن منو" data-label-close="بستن منو" @class([
        'group hidden size-11 flex-col items-center justify-center gap-[5px] rounded-full border-[1.5px] bg-transparent max-lg:flex',
        'border-night-line text-on-night' => $dark,
        'border-line text-ink' => ! $dark,
    ])>
        <span class="block h-0.5 w-5 rounded-full bg-current transition-transform motion-reduce:transition-none group-aria-expanded:translate-y-[7px] group-aria-expanded:rotate-45"></span>
        <span class="block h-0.5 w-5 rounded-full bg-current group-aria-expanded:opacity-0"></span>
        <span class="block h-0.5 w-5 rounded-full bg-current transition-transform motion-reduce:transition-none group-aria-expanded:-translate-y-[7px] group-aria-expanded:-rotate-45"></span>
    </button>
</header>
<x-layout.mobile-menu :links="$links" :variant="$variant" :login-url="$loginUrl" :download-href="$downloadHref"/>
