{{--
    PWA install metadata (L8-01), printed by <x-layout.assets/>. Data: $meta (App\Domain\Pwa\Data\InstallMetadata).
    Only link/meta tags: nothing render-blocking. The manifest is same-origin and needs no credentials.
--}}
@props(['meta'])
<link rel="manifest" href="{{ $meta->manifestUrl }}">
<meta name="theme-color" media="(prefers-color-scheme: light)" content="{{ $meta->themeColorLight }}">
<meta name="theme-color" media="(prefers-color-scheme: dark)" content="{{ $meta->themeColorDark }}">
<meta name="application-name" content="{{ $meta->applicationName }}">
<meta name="mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-title" content="{{ $meta->applicationName }}">
<meta name="apple-mobile-web-app-status-bar-style" content="default">
<link rel="icon" href="{{ $meta->faviconIco }}" sizes="32x32">
@if ($meta->faviconSvg !== null)
<link rel="icon" href="{{ $meta->faviconSvg }}" type="image/svg+xml">
@endif
@if ($meta->faviconPng !== null)
<link rel="icon" href="{{ $meta->faviconPng }}" type="image/png" sizes="32x32">
@endif
<link rel="apple-touch-icon" href="{{ $meta->appleTouchIcon }}" sizes="180x180">
<link rel="mask-icon" href="{{ $meta->maskIcon }}" color="{{ $meta->maskIconColor }}">
