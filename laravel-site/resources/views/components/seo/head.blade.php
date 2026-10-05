{{--
    Every SEO head tag of the page, decided by App\Domain\Seo\SeoManager (settings defaults → seo_meta → controller
    overrides). Used once by the layout: <x-seo.head/>. Pages never write head tags themselves.
    prev/next are intentionally not emitted (Google ignores them). JSON-LD: ONE @graph script built from the request's
    App\Domain\Seo\Schema\SchemaGraph (pages add nodes there, never their own scripts). The `seo.jsonld` stack is legacy.
--}}
@inject('seo', \App\Domain\Seo\SeoManager::class)
@inject('schemaGraph', \App\Domain\Seo\Schema\PageGraph::class)
@inject('headCode', \App\Domain\Seo\Indexing\HeadCode::class)
@php($head = $seo->resolve())
<title>{{ $head->title }}</title>
<meta name="description" content="{{ $head->description }}">
<link rel="canonical" href="{{ $head->canonical }}">
<meta name="robots" content="{{ $head->robots }}">
@foreach ($head->openGraph as $property => $content)
<meta property="{{ $property }}" content="{{ $content }}">
@endforeach
@foreach ($head->twitter as $name => $content)
<meta name="{{ $name }}" content="{{ $content }}">
@endforeach
@foreach ($head->verification as $name => $content)
<meta name="{{ $name }}" content="{{ $content }}">
@endforeach
{{-- Admin head code (L7-04): rebuilt from an allow-list of same-origin <meta>/<link> tags, never scripts. --}}
{!! $headCode->html() !!}
@foreach ($head->feeds as $feed)
<link rel="alternate" type="application/rss+xml" title="{{ $feed['title'] }}" href="{{ $feed['href'] }}">
@endforeach
{!! $schemaGraph->script($head) !!}
@stack('seo.jsonld')
{{ $slot ?? '' }}
