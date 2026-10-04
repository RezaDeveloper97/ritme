{{--
    Article page (L4-03, design/html/article.html): /blog/{slug}. Data: $page = App\Domain\Blog\Rendering\ArticlePage
    (ShowPostController → ArticlePageBuilder; cached reads only). Head tags + JSON-LD (BlogPosting, WebPage with
    reviewedBy, BreadcrumbList via <x-ui.breadcrumbs>) are set before the layout renders; `article:*` OG tags are
    pushed into the `head` stack here. The body is sanitised HTML from ArticleBodyRenderer (pictures, tables, ids).
--}}
@extends('layouts.app')

@php
    /** @var \App\Domain\Blog\Rendering\ArticlePage $page */
    $post = $page->post;
@endphp

@push('head')
    <meta property="article:published_time" content="{{ $post->publishedAt->setTimezone(config('app.timezone'))->toIso8601String() }}">
    <meta property="article:modified_time" content="{{ $post->updatedContentAt->setTimezone(config('app.timezone'))->toIso8601String() }}">
    @if ($post->category)
        <meta property="article:section" content="{{ $post->category->name }}">
    @endif
    @foreach ($post->tags as $tag)
        <meta property="article:tag" content="{{ $tag->name }}">
    @endforeach
    @if ($page->authorUrl)
        <meta property="article:author" content="{{ \App\Domain\Seo\Support\CanonicalUrl::normalize($page->authorUrl, (string) config('app.url')) }}">
    @endif
@endpush

@section('content')
    <article aria-labelledby="article-title">
        <header class="flex max-w-300 flex-col gap-4.5 px-30 pt-12 pb-0 max-lg:px-5 max-lg:pt-[26.4px]">
            {{-- Design shows «مجله / دسته» only: the current-page crumb stays in the JSON-LD trail, the h1 below names it. --}}
            <x-ui.breadcrumbs :items="$page->breadcrumbs" class="[&_li:last-child]:hidden"/>
            <h1 id="article-title" class="m-0 font-display text-[48px] leading-display font-normal text-ink max-sm:text-[30px]">{{ $post->title }}</h1>
            <x-blog.meta :post="$post"/>
        </header>

        <div class="flex items-start gap-16 px-30 pt-8 pb-20 max-lg:flex-wrap max-lg:px-5 max-lg:pb-11">
            <div class="flex max-w-190 min-w-0 flex-1 flex-col gap-5 max-lg:basis-75 max-sm:w-full max-sm:max-w-full max-sm:basis-full">
                <x-blog.cover :cover="$page->cover" :mobile="$page->mobileCover" :alt="$page->coverAlt" :stage="$post->lifeStage"/>
                <div class="rt-article">{!! $page->body->html !!}</div>
                <x-blog.sources :sources="$post->sources"/>
            </div>

            <aside aria-label="درباره این مقاله" class="flex w-80 shrink-0 flex-col gap-5 max-sm:w-full">
                <x-blog.toc :outline="$page->body->outline"/>
                <x-blog.app-card :stage="$post->lifeStage" :href="$page->downloadUrl"/>
                <x-blog.reviewer :post="$post" :author-url="$page->authorUrl" :reviewer-url="$page->reviewerUrl"/>
                <x-blog.share :links="$page->share" :url="$page->url"/>
                <x-blog.adjacent :previous="$page->previous" :next="$page->next" :category="$post->category"/>
            </aside>
        </div>
    </article>

    @if ($page->related !== [])
        <section aria-labelledby="related-title" class="flex flex-col gap-6 bg-surface px-30 pt-16 pb-24 max-lg:px-5 max-lg:pt-[35.2px] max-lg:pb-[52.8px]">
            <h2 id="related-title" class="m-0 font-display text-[30px] leading-heading font-normal text-ink">مقاله‌های مرتبط</h2>
            {{-- The design's card cover renders 180px + 2×18px padding (content-box); x-cards.article uses border-box h-45. --}}
            <div class="grid grid-cols-3 gap-4.5 max-sm:grid-cols-1 ">
                @foreach ($page->related as $related)
                    <x-cards.article :post="$related"/>
                @endforeach
            </div>
        </section>
    @endif
@endsection
