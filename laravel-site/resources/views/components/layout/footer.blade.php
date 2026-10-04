{{--
    Footer markup, rendered (and fragment-cached) by App\View\Components\Layout\Footer — use <x-layout.footer/>.
    Data: $homeUrl, $siteName, $tagline, $columns (list<FooterColumnData>), $stores / $socials (list<LinkData>,
    empty settings already dropped), $enamadHtml (sanitised) / $enamadCode, $year, $emergency, $emergencyLabel.
--}}
<footer class="col-start-1 row-start-5 flex flex-col gap-10 bg-night px-30 pt-16 pb-8 text-on-night max-lg:px-5 max-lg:pt-[35.2px]">
    <div class="grid grid-cols-[1.3fr_repeat(5,minmax(0,1fr))] gap-7 max-lg:grid-cols-3 max-sm:grid-cols-1">
        <div class="flex min-w-0 flex-col gap-3.5 max-lg:col-span-full">
            <a href="{{ $homeUrl }}" class="flex items-center gap-2.5 text-on-night hover:text-on-night">
                <span class="flex size-10 items-center justify-center rounded-full bg-lilac"><x-icon name="drop" class="size-5 text-white"/></span>
                <span class="font-display text-[30px] leading-none">{{ $siteName }}</span>
            </a>
            @if ($tagline !== '')
                <p class="m-0 text-base leading-loose text-on-night-muted">{{ $tagline }}</p>
            @endif
            @if ($stores !== [])
                <ul class="flex origin-top-right scale-85 flex-wrap gap-2.5 text-base max-sm:scale-none" aria-label="دریافت اپ">
                    @foreach ($stores as $store)
                        <li><a href="{{ $store->url }}" rel="noopener" class="box-content flex h-13 items-center gap-2.5 rounded-lg border border-night-line bg-night-card px-4.5 text-base font-extrabold text-on-night hover:text-on-night">
                            <x-icon name="download" class="size-4.5"/>
                            <span class="flex flex-col leading-snug"><span class="text-2xs font-semibold opacity-75">دریافت از</span>{{ $store->label }}</span>
                        </a></li>
                    @endforeach
                </ul>
            @endif
        </div>
        @foreach ($columns as $column)
            <nav aria-label="{{ $column->title }}" class="flex min-w-0 flex-col gap-3">
                <p class="m-0 text-md font-bold text-on-night">{{ $column->title }}</p>
                <ul class="flex flex-col gap-3 text-base">
                    @foreach ($column->links as $link)
                        <li><a href="{{ $link->url }}" class="font-semibold text-on-night-muted hover:text-on-night">{{ $link->label }}</a></li>
                    @endforeach
                    @if ($loop->last && ($enamadHtml !== '' || $enamadCode !== null))
                        <li class="text-base font-semibold text-on-night-muted [&_a]:text-on-night-muted [&_a:hover]:text-on-night">
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
    <div class="flex items-center justify-between border-t border-night-line pt-6 text-sm text-on-night-muted max-lg:flex-wrap max-sm:flex-col max-sm:items-start max-sm:gap-3">
        <p class="m-0">© {{ $year }} {{ $siteName }} · {{ $siteName }} جایگزین اورژانس نیست؛ در شرایط اضطراری با <a href="tel:{{ $emergency }}" class="text-on-night-muted underline-offset-2 hover:text-on-night hover:underline">{{ $emergencyLabel }}</a> تماس بگیر.</p>
        @if ($socials !== [])
            <ul class="flex gap-4.5 text-sm" aria-label="شبکه‌های اجتماعی">
                @foreach ($socials as $social)
                    <li><a href="{{ $social->url }}" rel="me noopener" class="text-on-night-muted hover:text-on-night">{{ $social->label }}</a></li>
                @endforeach
            </ul>
        @endif
    </div>
</footer>
