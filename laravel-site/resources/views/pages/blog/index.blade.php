{{--
    Magazine list (L4-02, design/html/blog.html): /blog, /blog/category/{slug}, /blog/tag/{slug}, /blog/author/{slug}.
    Data (BlogListingController, DTOs only): $intro [eyebrow, title, lead], $chips (chip-nav items), $featured
    (?PostCardData, /blog page 1), $featuredReviewer, $author (?AuthorData), $breadcrumbs (?list<BreadcrumbItem>, tag +
    author pages show the trail), $postsHeading, $posts (list<PostCardData>), $pagination, $newsletterSource,
    $appLinks, $qrUrl. JSON-LD (CollectionPage/ProfilePage, ItemList, BreadcrumbList) is registered by the controller.
--}}
@extends('layouts.app')

@section('content')
    <x-ui.page-intro :eyebrow="$intro['eyebrow']" :title="$intro['title']" :lead="$intro['lead']">
        @if ($author)
            @include('pages.blog.partials.author', ['author' => $author])
        @endif
        @if ($chips !== [])
            <x-ui.chip-nav :label="__('blog.chips_label')" :items="$chips"/>
        @endif
        @if ($breadcrumbs)
            <x-ui.breadcrumbs :items="$breadcrumbs" class="order-first"/>
        @endif
    </x-ui.page-intro>

    @if ($featured)
        <section class="flex flex-col gap-10 px-30 pt-6 pb-12 max-lg:px-5 max-lg:pb-[26.4px]">
            {{-- Design keeps the text block's 32px top padding when the card wraps (x-cards.article drops it). --}}
            <x-cards.article :post="$featured" featured as="h2" :reviewer="$featuredReviewer" class="max-lg:[&>span:last-child]:pt-8"/>
        </section>
    @endif

    <section aria-labelledby="posts-title" @class(['flex flex-col gap-4.5 px-30 pb-12 max-lg:px-5 max-lg:pb-[26.4px]', 'pt-0' => $featured, 'pt-6' => ! $featured])>
        <h2 id="posts-title" class="sr-only">{{ $postsHeading }}</h2>
        @if ($posts === [])
            <p class="m-0 rounded-4xl border border-line bg-surface p-8 text-lg leading-loose font-medium text-muted">{{ __('blog.empty') }}</p>
        @else
            {{-- The design's card cover renders 180px + 2×18px padding (content-box); x-cards.article uses border-box h-45. --}}
            <div class="grid grid-cols-3 gap-4.5 max-sm:grid-cols-1 [&_a>span:first-child]:h-54">
                @foreach ($posts as $post)
                    <x-cards.article :post="$post"/>
                @endforeach
            </div>
        @endif
        @if ($pagination)
            @include('pages.blog.partials.pagination', ['pagination' => $pagination])
        @endif
    </section>

    <div class="flex flex-col gap-3 px-30 pt-0 pb-24 max-lg:px-5 max-lg:pb-[52.8px]">
        <x-ui.newsletter id="newsletter" name="email" :action="route('newsletter.store')"
                         :title="__('blog.newsletter.title')" :text="__('blog.newsletter.text')"
                         :placeholder="__('blog.newsletter.placeholder')" :button="__('blog.newsletter.button')">
            <input type="hidden" name="source" value="{{ $newsletterSource }}">
            <div aria-hidden="true" class="sr-only">
                <label for="newsletter-website">{{ __('blog.newsletter.honeypot') }}</label>
                <input id="newsletter-website" type="text" name="{{ \App\Http\Controllers\Blog\NewsletterController::HONEYPOT }}" value="" tabindex="-1" autocomplete="off">
            </div>
        </x-ui.newsletter>
        @if (session('newsletter_status'))
            <p role="status" class="m-0 flex items-center gap-2 rounded-3xl bg-success-soft px-5 py-3.5 text-base font-bold text-success">
                <x-icon name="check" class="size-4.5 shrink-0"/>{{ session('newsletter_status') }}
            </p>
        @elseif ($errors->newsletter->has('email'))
            <p role="alert" class="m-0 flex items-center gap-2 rounded-3xl bg-danger-soft px-5 py-3.5 text-base font-bold text-danger">
                <x-icon name="alert-triangle" class="size-4.5 shrink-0"/>{{ $errors->newsletter->first('email') }}
            </p>
        @endif
    </div>

    <x-ui.app-cta :title="__('blog.app_cta.title')" :lead="__('blog.app_cta.lead')" :links="$appLinks" :qr-url="$qrUrl"/>
@endsection
