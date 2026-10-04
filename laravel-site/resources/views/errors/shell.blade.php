{{--
    Status page on the site layout: the 404/410/419/429 error views and the L1-05 route placeholders.
    Data: $title (the single h1), $message, optional $code (HTTP status; error views), optional $breadcrumbs
    (list<BreadcrumbItem>), optional $appCta. Always noindex: error views set it here, the placeholder controller
    sets it itself. Helpful links come from the StaticPage registry; the search link appears once the `search`
    route exists (L4-04). 500/503 use errors.standalone instead (they must render without the database).
--}}
@inject('statusSeo', \App\Domain\Seo\SeoManager::class)
@inject('statusNav', \App\Domain\Content\SiteNavigation::class)
@php
    if (isset($code)) {
        $statusSeo->title($title)->noindex();
    }
    $statusLinks = array_map(
        static fn (\App\Domain\Content\Enums\StaticPage $page): array => ['label' => $page->label(), 'url' => $statusNav->url($page)],
        [...\App\Domain\Content\Enums\StaticPage::stages(), \App\Domain\Content\Enums\StaticPage::Blog, \App\Domain\Content\Enums\StaticPage::Tools, \App\Domain\Content\Enums\StaticPage::Faq, \App\Domain\Content\Enums\StaticPage::Contact],
    );
    $statusSearch = \Illuminate\Support\Facades\Route::has('search') ? route('search', [], false) : null;
@endphp
@extends('layouts.app', ['appCta' => $appCta ?? false])

@section('content')
    <x-ui.section pad="lg">
        @if (! empty($breadcrumbs))
            <x-ui.breadcrumbs :items="$breadcrumbs" class="mb-6"/>
        @endif
        @isset($code)
            <p class="font-display text-d-md leading-none text-primary">{{ fa_digits($code) }}</p>
        @endisset
        <h1 class="mt-4 font-display text-d-xl leading-display text-ink">{{ $title }}</h1>
        <p class="mt-4 max-w-[640px] text-xl leading-relaxed text-muted">{{ $message }}</p>
        <div class="mt-8 flex flex-wrap items-center gap-3">
            <x-ui.button :href="$statusNav->url(\App\Domain\Content\Enums\StaticPage::Home)" size="lg" icon-end="arrow-left">صفحه اصلی</x-ui.button>
            @if ($statusSearch !== null)
                <x-ui.button :href="$statusSearch" variant="outline" size="lg">جست‌وجو در ریتمی</x-ui.button>
            @endif
        </div>
        <h2 class="mt-14 text-3xl font-extrabold text-ink">صفحه‌های پربازدید</h2>
        <ul class="mt-5 grid grid-cols-3 gap-3 max-lg:grid-cols-2 max-sm:grid-cols-1">
            @foreach ($statusLinks as $link)
                <li>
                    <a href="{{ $link['url'] }}" class="flex items-center justify-between gap-3 rounded-2xl border border-line bg-surface px-5 py-4 text-md font-bold text-ink hover:border-primary hover:text-primary">
                        <span>{{ $link['label'] }}</span>
                        <x-icon name="arrow-left" class="size-4.5 text-primary"/>
                    </a>
                </li>
            @endforeach
        </ul>
    </x-ui.section>
@endsection
