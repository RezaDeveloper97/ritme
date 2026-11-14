{{--
    Footer markup, rendered (and fragment-cached) by App\View\Components\Layout\Footer — use <x-layout.footer/>.
    Data: $homeUrl, $siteName, $logoUrl, $tagline, $columns (list<FooterColumnData>), $stores / $socials (list<LinkData>,
    empty settings already dropped), $enamadSeal (?EnamadSeal: official seal image) else $enamadHtml (sanitised) / $enamadCode, $year, $emergency, $emergencyLabel.
    Nov 2026 redesign: ink background, brand + tagline, three link columns (مرحله‌ها · ریتمی · اعتماد); store links
    live in each page's #download band, so they are not repeated here.
--}}
<footer class="bleed col-start-1 row-start-5 flex flex-col gap-10 bg-ink px-30 pt-16 pb-8 text-white max-xl:px-6 max-lg:px-5 max-lg:pt-12">
    <div class="grid grid-cols-[1.6fr_repeat(3,minmax(0,1fr))] gap-8 max-lg:grid-cols-3 max-sm:grid-cols-2">
        <div class="flex max-w-80 min-w-0 flex-col gap-3.5 max-lg:col-span-full">
            <a href="{{ $homeUrl }}" class="flex items-center gap-2 text-white hover:text-white">
                <img src="{{ $logoUrl }}" width="106" height="40" alt="{{ $siteName }}" class="h-10 w-auto shrink-0" decoding="async">
            </a>
            @if ($tagline !== '')
                <p class="m-0 text-base leading-loose text-on-ink-muted">{{ $tagline }}</p>
            @endif
        </div>
        @foreach ($columns as $column)
            <nav aria-label="{{ $column->title }}" class="flex min-w-0 flex-col gap-3">
                <p class="m-0 text-md font-bold text-white">{{ $column->title }}</p>
                <ul class="flex flex-col gap-3 text-base">
                    @foreach ($column->links as $link)
                        <li><a href="{{ $link->url }}" class="font-semibold text-on-ink-muted hover:text-white">{{ $link->label }}</a></li>
                    @endforeach
                    @if ($loop->last && $enamadSeal !== null)
                        <li class="mt-1">
                            <a href="{{ $enamadSeal->linkUrl() }}" target="_blank" rel="noopener nofollow" referrerpolicy="origin" class="inline-flex rounded-xl bg-white p-1.5" aria-label="نماد اعتماد الکترونیکی">
                                <img src="{{ $enamadSeal->imageUrl() }}" width="{{ $enamadSeal::WIDTH }}" height="{{ $enamadSeal::HEIGHT }}" alt="نماد اعتماد الکترونیکی" referrerpolicy="origin" loading="lazy" decoding="async" class="h-22 w-auto">
                            </a>
                        </li>
                    @elseif ($loop->last && ($enamadHtml !== '' || $enamadCode !== null))
                        <li class="text-base font-semibold text-on-ink-muted [&_a]:text-on-ink-muted [&_a:hover]:text-white">
                            @if ($enamadHtml !== '')
                                {!! $enamadHtml !!}
                            @else
                                <span>{{ $enamadCode }}</span>
                            @endif
                        </li>
                    @endif
                </ul>
            </nav>
        @endforeach
    </div>
    <div class="flex items-center justify-between gap-4 border-t border-night-line pt-6 text-sm leading-relaxed text-on-ink-muted max-sm:flex-col max-sm:items-start">
        <p class="m-0">© {{ $year }} {{ $siteName }}</p>
        @if ($socials !== [])
            <ul class="flex gap-4.5 text-sm" aria-label="شبکه‌های اجتماعی">
                @foreach ($socials as $social)
                    <li><a href="{{ $social->url }}" rel="me noopener" class="text-on-ink-muted hover:text-white">{{ $social->label }}</a></li>
                @endforeach
            </ul>
        @endif
    </div>
</footer>
