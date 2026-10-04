{{--
    <x-blog.meta :post="$post"/> — the line under the article h1 (design: «۶ دقیقه مطالعه · به‌روزرسانی: … ·
    بازبینی علمی: …»). Dates in Jalali with Persian digits inside <time>.
--}}
@props(['post'])
@php
    /** @var \App\Domain\Blog\Data\PostData $post */
    $reviewer = $post->reviewer;
@endphp
<div {{ $attributes->class('flex items-center gap-4.5 text-base font-bold text-muted max-lg:flex-wrap') }}>
    <span>{{ fa_digits($post->readingTime) }} دقیقه مطالعه</span>
    <span aria-hidden="true">·</span>
    <span>به‌روزرسانی: <time datetime="{{ $post->updatedContentAt->format(DATE_ATOM) }}">{{ jdate($post->updatedContentAt, 'j F Y') }}</time></span>
    @if ($reviewer)
        <span aria-hidden="true">·</span>
        <span class="flex items-center gap-1.5">
            <x-icon name="shield-check" class="size-4 shrink-0 text-stage-postpartum"/>
            بازبینی علمی: {{ $reviewer->name }}@if ($reviewer->credentials)، {{ $reviewer->credentials }}@endif
        </span>
    @endif
</div>
