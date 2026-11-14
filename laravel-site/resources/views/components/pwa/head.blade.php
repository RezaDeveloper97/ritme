{{--
    PWA install metadata (L8-01), printed by <x-layout.assets/>. Data: $meta (App\Domain\Pwa\Data\InstallMetadata).
    Only link/meta tags: nothing render-blocking. No <link rel="manifest"> and no *-web-app-capable metas on purpose:
    this is the marketing site, not the app, so browsers must not offer «Install app» for it.
    No mask-icon either: Safari would show that one-colour mark instead of the Ritme logo favicon.
--}}
@props(['meta'])
<meta name="theme-color" media="(prefers-color-scheme: light)" content="{{ $meta->themeColorLight }}">
<meta name="theme-color" media="(prefers-color-scheme: dark)" content="{{ $meta->themeColorDark }}">
<meta name="application-name" content="{{ $meta->applicationName }}">
<link rel="icon" href="{{ $meta->faviconIco }}" sizes="32x32">
@if ($meta->faviconSvg !== null)
<link rel="icon" href="{{ $meta->faviconSvg }}" type="image/svg+xml">
@endif
@if ($meta->faviconPng !== null)
<link rel="icon" href="{{ $meta->faviconPng }}" type="image/png" sizes="32x32">
@endif
<link rel="apple-touch-icon" href="{{ $meta->appleTouchIcon }}" sizes="180x180">
