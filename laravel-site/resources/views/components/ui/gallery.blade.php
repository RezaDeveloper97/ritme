{{--
    <x-ui.gallery variant="mosaic" :images="[$cover, $img2, …]" alt="استخر آب‌پری" :total="12" more-href="#photos"/>
    <x-ui.gallery variant="product" :images="[…]" alt="بادی آستین‌بلند"/>
    Gallery shell (AUDIT §2.2; lightboxes come with L5-03 / L6-03). `images`: MediaData|int ids rendered through
    <x-picture> (the component never queries; ids are resolved by Picture's cached repository). Missing images render
    a decorative lavender tile, so the layout holds with design placeholders.
    mosaic: `2fr 1fr 1fr`, two 220px rows, first image spans both, «همه N عکس» link. product: main image + 96px thumbs.
--}}
@props(['images' => [], 'variant' => 'mosaic', 'alt' => '', 'total' => null, 'moreHref' => null])
@php($images = array_values($images))
@if ($variant === 'product')
<div {{ $attributes->class('flex flex-col gap-3') }}>
    <div class="aspect-square overflow-hidden rounded-6xl bg-lavender">
        @if ($images[0] ?? null)
            <x-picture :media="$images[0]" :alt="$alt" sizes="(max-width: 768px) 100vw, 50vw" class="size-full object-cover" priority/>
        @endif
    </div>
    @if (count($images) > 1)
        <ul class="m-0 flex list-none gap-2.5 p-0" aria-label="تصاویر دیگر">
            @foreach (array_slice($images, 1, 5) as $image)
                <li class="size-24 overflow-hidden rounded-2xl border-[1.5px] border-line bg-lavender">
                    @if ($image)<x-picture :media="$image" :alt="$alt.' — تصویر '.fa_digits($loop->iteration + 1)" sizes="96px" class="size-full object-cover"/>@endif
                </li>
            @endforeach
        </ul>
    @endif
</div>
@else
<div {{ $attributes->class('relative grid h-fit grid-cols-[2fr_1fr_1fr] grid-rows-[220px_220px] gap-2.5 overflow-hidden rounded-6xl max-sm:grid-cols-1 max-sm:grid-rows-[240px]') }}>
    @for ($i = 0; $i < 5; $i++)
        <div @class(['overflow-hidden bg-lavender', 'row-span-2 max-sm:row-span-1' => $i === 0, 'max-sm:hidden' => $i > 0])>
            @if ($images[$i] ?? null)
                <x-picture :media="$images[$i]" :alt="$i === 0 ? $alt : $alt.' — تصویر '.fa_digits($i + 1)" :sizes="$i === 0 ? '(max-width: 700px) 100vw, 50vw' : '25vw'" class="size-full object-cover" :priority="$i === 0"/>
            @endif
        </div>
    @endfor
    @if ($moreHref)
        <a href="{{ $moreHref }}" class="absolute end-4.5 bottom-4.5 flex h-11 items-center gap-2 rounded-full border border-line bg-surface px-4.5 text-base font-extrabold text-ink hover:text-ink">
            <x-icon name="camera" class="size-4.5 text-primary"/>همه {{ fa_digits((int) ($total ?? count($images))) }} عکس
        </a>
    @endif
</div>
@endif
