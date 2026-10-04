{{--
    A body image of an article, rendered by App\Domain\Blog\Rendering\ArticleBodyRenderer for every
    `<img data-media-id>` (view data: media id, alt|null, eager, sizes). Not used as a tag. Lazy by default; `eager`
    (first image of a post without cover — a possible LCP) loads eagerly with high priority but pushes no preload,
    so the rendered body can be cached. Missing media renders nothing.
--}}
<x-picture :media="$media" :alt="$alt" :sizes="$sizes"
           :loading="$eager ? 'eager' : null" :fetchpriority="$eager ? 'high' : null"
           class="h-auto w-full rounded-4xl"/>
