{{--
    <x-ui.qr url="https://…" label="اسکن و دانلود" :size="130"/>
    A real QR code of the app link (the design fakes one with a conic gradient — AUDIT §2.2). Rendered server-side as
    SVG by the local bacon/bacon-qr-code library and cached forever in the `media` cache-aside namespace keyed by the
    URL — never an external QR service. The modules use `currentColor` (night) on a white 180px tile. Renders nothing
    for an empty URL.
--}}
@props(['url' => null, 'label' => 'اسکن و دانلود', 'size' => 130])
@php
    $qrSvg = '';
    if (is_string($url) && $url !== '') {
        $qrSvg = app(\App\Support\Cache\CacheAside::class)->rememberForever(
            \App\Support\Cache\CacheKey::make('media', 'qr', (string) $size, sha1($url)),
            static function () use ($url, $size): string {
                $writer = new \BaconQrCode\Writer(new \BaconQrCode\Renderer\ImageRenderer(
                    new \BaconQrCode\Renderer\RendererStyle\RendererStyle((int) $size, 0),
                    new \BaconQrCode\Renderer\Image\SvgImageBackEnd(),
                ));
                $svg = $writer->writeString($url);
                $svg = (string) preg_replace(['~<\?xml[^>]*\?>\s*~', '~<rect[^>]*fill="#f{3,6}"[^>]*/>~i'], '', $svg);
                $svg = (string) preg_replace('~fill="#[0-9a-f]{3,8}"~i', 'fill="currentColor"', $svg);
                // Bacon paths have no fill attribute: inherit currentColor from the root.
                return (string) preg_replace('~<svg ~', '<svg fill="currentColor" aria-hidden="true" focusable="false" class="block size-full" ', $svg, 1);
            },
        );
    }
@endphp
@if ($qrSvg !== '')
<figure {{ $attributes->class('m-0 flex size-45 shrink-0 flex-col items-center justify-center gap-1.5 rounded-4xl bg-surface text-night') }}>
    <span class="block size-32.5" role="img" aria-label="کد QR لینک دریافت اپ">{!! $qrSvg !!}</span>
    <figcaption class="text-xs font-extrabold text-ink">{{ $label }}</figcaption>
</figure>
@endif
