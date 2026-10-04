{{--
    <x-blog.share :links="$page->share" :url="$page->url"/> — share as plain links (no third-party script; nothing
    is requested until the reader clicks) + the article link in a read-only field to copy. Without JS the field is
    enough (select + copy by hand); with JS the `share` data-module reveals the «کپی پیوند» button (hidden in the
    HTML) and announces the result in the status line. Copy: lang blog.article.share.*.
--}}
@props(['links' => [], 'url'])
<section aria-labelledby="article-share-title" data-module="share"
         data-share-copied="{{ __('blog.article.share.copied') }}" data-share-failed="{{ __('blog.article.share.failed') }}"
         {{ $attributes->class('flex flex-col gap-3') }}>
    <h2 id="article-share-title" class="m-0 flex items-center gap-2 font-sans text-md leading-normal font-bold text-ink">
        <x-icon name="share" class="size-4.5 text-primary"/>{{ __('blog.article.share.title') }}
    </h2>
    <ul class="m-0 flex list-none flex-wrap gap-2 p-0">
        @foreach ($links as $link)
            <li>
                <a href="{{ $link['href'] }}" target="_blank" rel="noopener noreferrer nofollow"
                   class="inline-flex h-10 items-center rounded-full border border-line bg-surface px-4 text-sm-plus font-bold text-ink hover:border-primary hover:text-primary">{{ $link['label'] }}<span class="sr-only"> {{ __('blog.article.share.new_window') }}</span></a>
            </li>
        @endforeach
    </ul>
    <label for="article-share-url" class="text-sm font-bold text-muted">{{ __('blog.article.share.url_label') }}</label>
    <div class="flex items-center gap-2">
        <input id="article-share-url" type="url" readonly value="{{ $url }}" dir="ltr" data-share-url
               class="h-11 w-full min-w-0 flex-1 rounded-xl border border-line bg-surface px-3.5 text-sm text-ink">
        <button type="button" hidden data-share-copy aria-describedby="article-share-status"
                class="inline-flex h-11 shrink-0 items-center gap-1.5 rounded-xl border border-primary bg-primary px-4 text-sm font-bold text-white transition-colors hover:bg-primary-hover">
            {{ __('blog.article.share.copy') }}
        </button>
    </div>
    <p id="article-share-status" role="status" aria-live="polite" data-share-status class="m-0 text-sm font-semibold text-muted empty:hidden"></p>
</section>
