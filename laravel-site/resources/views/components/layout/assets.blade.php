{{-- Rendered by App\View\Components\Layout\Assets: critical CSS (L9-01), above-the-fold font preloads, Vite entries, PWA install tags. --}}
@if ($criticalCss !== null)
<link rel="expect" href="#main-end" blocking="render">
<style>{!! $criticalCss !!}</style>
@endif
@foreach ($fonts as $font)
<link rel="preload" href="{{ $font }}" as="font" type="font/woff2" crossorigin>
@endforeach
@if ($criticalCss !== null)
<link rel="preload" href="{{ $stylesheet }}" as="style">
@vite([\App\View\Components\Layout\Assets::SCRIPT])
@foreach ($modules as $module)
<link rel="modulepreload" href="{{ $module }}">
@endforeach
@push('deferred-styles')
<link rel="stylesheet" href="{{ $stylesheet }}">
@endpush
@else
@vite([\App\View\Components\Layout\Assets::STYLESHEET, \App\View\Components\Layout\Assets::SCRIPT])
@endif
<x-pwa.head :meta="$pwa"/>
