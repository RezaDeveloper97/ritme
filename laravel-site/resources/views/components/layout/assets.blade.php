{{-- Rendered by App\View\Components\Layout\Assets: theme-color, above-the-fold font preloads, Vite entries. --}}
@if ($themeColor !== null)
<meta name="theme-color" content="{{ $themeColor }}">
@endif
@foreach ($fonts as $font)
<link rel="preload" href="{{ $font }}" as="font" type="font/woff2" crossorigin>
@endforeach
@vite(['resources/css/app.css', 'resources/js/app.js'])
