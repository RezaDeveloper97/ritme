{{-- <x-picture> — markup only; every value is resolved in App\View\Components\Picture (no queries here). --}}
@if ($preloads !== [])
    @pushOnce('head', $preloadKey)
        @foreach ($preloads as $preload)
            <link rel="preload" as="image" @if ($preload['href']) href="{{ $preload['href'] }}" @endif @if ($preload['imagesrcset']) imagesrcset="{{ $preload['imagesrcset'] }}" imagesizes="{{ $preload['imagesizes'] }}" @endif type="{{ $preload['type'] }}" @if ($preload['media']) media="{{ $preload['media'] }}" @endif fetchpriority="high">
        @endforeach
    @endPushOnce
@endif
<picture @if ($pictureClass) class="{{ $pictureClass }}" @endif>
    @foreach ($sources as $source)
        <source @if ($source['media']) media="{{ $source['media'] }}" @endif type="{{ $source['type'] }}" srcset="{{ $source['srcset'] }}" sizes="{{ $sizes }}" @if ($source['width'] && $source['height']) width="{{ $source['width'] }}" height="{{ $source['height'] }}" @endif>
    @endforeach
    <img {{ $attributes->except(['src', 'srcset', 'sizes', 'width', 'height', 'alt', 'loading', 'fetchpriority', 'decoding', 'style'])->class([$placeholderClass]) }} src="{{ $src }}" @if ($srcset) srcset="{{ $srcset }}" sizes="{{ $sizes }}" @endif @if ($imgWidth && $imgHeight) width="{{ $imgWidth }}" height="{{ $imgHeight }}" @endif alt="{{ $altText }}" loading="{{ $loadingValue }}" @if ($fetchpriorityValue) fetchpriority="{{ $fetchpriorityValue }}" @endif decoding="{{ $decodingValue }}">
</picture>
