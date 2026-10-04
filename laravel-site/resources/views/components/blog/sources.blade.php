{{--
    <x-blog.sources :sources="$post->sources"/> — «منابع: …» + the medical disclaimer (design: top border, 14px muted).
    `sources` is HTML sanitised on save (PostContent: allow-list, external links rel="noopener").
--}}
@props(['sources' => null])
<div {{ $attributes->class('flex flex-col border-t border-t-line pt-5 text-base leading-loose text-muted') }}>
    @if ($sources)
        <div class="flex flex-wrap items-baseline gap-x-1.5">
            <span>{{ __('blog.article.sources') }}</span>
            <div class="rt-sources">{!! $sources !!}</div>
        </div>
    @endif
    <x-blog.disclaimer/>
</div>
