{{--
    <x-blog.meta :post="$post"/> — the line under the article h1 (design: «۶ دقیقه مطالعه · به‌روزرسانی: … ·
    بازبینی علمی: …»). Dates in Jalali with Persian digits inside <time>. Copy: lang blog.article.*.
    It also carries the article's view beacon (L4-03b, `beacon` default true): `data-module="view-beacon"` posts to
    `blog.view` once per post per tab session, so page-cache HITs are counted too (no inline JS; same origin).
--}}
@props(['post', 'beacon' => true])
@php
    /** @var \App\Domain\Blog\Data\PostData $post */
    $reviewer = $post->reviewer;
@endphp
<div {{ $attributes->class('flex items-center gap-4.5 text-base font-bold text-muted max-lg:flex-wrap') }}
     @if ($beacon) data-module="view-beacon" data-view-url="{{ route('blog.view', $post->slug) }}" data-view-id="{{ $post->id }}" @endif>
    <span>{{ __('blog.article.minutes', ['minutes' => fa_digits($post->readingTime)]) }}</span>
    <span aria-hidden="true">·</span>
    <span>{{ __('blog.article.updated') }} <time datetime="{{ $post->updatedContentAt->format(DATE_ATOM) }}">{{ jdate($post->updatedContentAt, 'j F Y') }}</time></span>
    @if ($reviewer)
        <span aria-hidden="true">·</span>
        <span class="flex items-center gap-1.5">
            <x-icon name="shield-check" class="size-4 shrink-0 text-stage-postpartum"/>
            {{ __('blog.article.reviewed_by') }} {{ $reviewer->name }}@if ($reviewer->credentials)، {{ $reviewer->credentials }}@endif
        </span>
    @endif
</div>
