{{-- Rendered by App\View\Components\Layout\Assets: above-the-fold font preloads, Vite entries, PWA install tags. --}}
@foreach ($fonts as $font)
<link rel="preload" href="{{ $font }}" as="font" type="font/woff2" crossorigin>
@endforeach
@vite(['resources/css/app.css', 'resources/js/app.js'])
<x-pwa.head :meta="$pwa"/>
