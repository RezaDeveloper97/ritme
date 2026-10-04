{{--
    <x-blog.share :links="$page->share" :url="$page->url"/> — share as plain links (no third-party script; nothing
    is requested until the reader clicks) + the article link in a read-only field to copy (works without JS; a
    one-click copy button needs a `share` data-module, see the L4-03 open items).
--}}
@props(['links' => [], 'url'])
<section aria-labelledby="article-share-title" {{ $attributes->class('flex flex-col gap-3') }}>
    <h2 id="article-share-title" class="m-0 flex items-center gap-2 font-sans text-md leading-normal font-bold text-ink">
        <x-icon name="share" class="size-4.5 text-primary"/>اشتراک‌گذاری
    </h2>
    <ul class="m-0 flex list-none flex-wrap gap-2 p-0">
        @foreach ($links as $link)
            <li>
                <a href="{{ $link['href'] }}" target="_blank" rel="noopener noreferrer nofollow"
                   class="inline-flex h-10 items-center rounded-full border border-line bg-surface px-4 text-sm-plus font-bold text-ink hover:border-primary hover:text-primary">{{ $link['label'] }}<span class="sr-only"> (پنجره تازه)</span></a>
            </li>
        @endforeach
    </ul>
    <label for="article-share-url" class="text-sm font-bold text-muted">پیوند مقاله برای کپی</label>
    <input id="article-share-url" type="url" readonly value="{{ $url }}" dir="ltr"
           class="h-11 w-full rounded-xl border border-line bg-surface px-3.5 text-sm text-ink">
</section>
