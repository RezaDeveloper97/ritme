{{-- Rendered by App\View\Components\Layout\Assets: critical CSS (L9-01), above-the-fold font preloads, Vite entries, PWA install tags. --}}
{{-- L9-02: the page-module preloads are fetchpriority="low" (they only enhance; they must not compete with the
     document, the preloaded fonts and the LCP image). The full stylesheet keeps its normal priority: a template whose
     critical CSS was cut from another URL (shop.category uses /shop's) still relies on it arriving before the paint. --}}
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
<link rel="modulepreload" href="{{ $module }}" fetchpriority="low">
@endforeach
@push('deferred-styles')
<link rel="stylesheet" href="{{ $stylesheet }}">
@endpush
@else
@vite([\App\View\Components\Layout\Assets::STYLESHEET, \App\View\Components\Layout\Assets::SCRIPT])
@endif
<x-pwa.head :meta="$pwa"/>
