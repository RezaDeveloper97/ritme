{{--
    Site search (L4-04): /search?q= — a plain GET form (works without JS), results of every SearchProvider.
    Data (SearchController): $results (SearchResults: query, hits list<SearchHit>, total, page), $heading (the h1),
    $tooShort (query given but no usable token), $count (Persian total), $pagination (blog pagination shape or null).
    Always noindex,follow; never page-cached. Copy: lang/fa/search.php.
--}}
@extends('layouts.app')

@section('content')
    <x-ui.page-intro :eyebrow="__('search.eyebrow')" :title="$heading" :lead="$results->searched() ? null : __('search.lead')" size="md">
        <form method="get" action="{{ route('search') }}" role="search" class="flex max-w-205 items-stretch gap-3 max-sm:flex-col">
            <label for="search-q" class="sr-only">{{ __('search.label') }}</label>
            <x-ui.form.input id="search-q" name="q" type="search" :value="$results->query" maxlength="100"
                             :placeholder="__('search.placeholder')" autocomplete="off" enterkeyhint="search" class="flex-1"/>
            <x-ui.button type="submit" size="2xl" icon="search">{{ __('search.button') }}</x-ui.button>
        </form>
    </x-ui.page-intro>

    <section aria-labelledby="search-results-title" class="flex flex-col gap-4.5 px-30 pt-6 pb-24 max-lg:px-5 max-lg:pb-[52.8px]">
        <h2 id="search-results-title" class="sr-only">{{ __('search.results_heading') }}</h2>

        @if ($tooShort)
            <p class="m-0 rounded-4xl border border-line bg-surface p-8 text-lg leading-loose font-medium text-muted">{{ __('search.too_short') }}</p>
        @elseif ($results->searched() && $results->total === 0)
            <div class="flex flex-col gap-4 rounded-4xl border border-line bg-surface p-8">
                <p class="m-0 text-lg leading-loose font-medium text-muted">{{ __('search.empty', ['query' => $results->query]) }}</p>
                <div class="flex flex-wrap gap-3">
                    <x-ui.button :href="route('blog.index')" variant="outline" size="sm">{{ __('search.browse_blog') }}</x-ui.button>
                    @if (Route::has('faq'))
                        <x-ui.button :href="route('faq')" variant="outline" size="sm">{{ __('search.browse_faq') }}</x-ui.button>
                    @endif
                </div>
            </div>
        @elseif ($results->searched())
            <p role="status" class="m-0 text-base font-bold text-muted">{{ __('search.count', ['count' => $count]) }}</p>
            <ol class="m-0 flex list-none flex-col gap-3 p-0">
                @foreach ($results->hits as $hit)
                    <li data-type="{{ $hit->type }}" class="flex flex-col gap-2 rounded-3xl border border-line bg-surface p-6 transition-colors hover:border-primary">
                        <span class="text-sm font-extrabold text-primary">{{ $hit->typeLabel }}</span>
                        <h3 class="m-0 text-lg leading-[1.7] font-extrabold text-ink">
                            <a href="{{ $hit->url }}" class="text-ink hover:text-primary">{{ $hit->title }}</a>
                        </h3>
                        @if ($hit->snippet)
                            <p class="m-0 text-base leading-loose font-medium text-muted">{{ $hit->snippet }}</p>
                        @endif
                    </li>
                @endforeach
            </ol>
            @if ($pagination)
                @include('pages.blog.partials.pagination', ['pagination' => $pagination])
            @endif
        @endif
    </section>
@endsection
