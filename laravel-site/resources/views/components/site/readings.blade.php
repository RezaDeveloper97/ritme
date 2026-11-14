{{--
    <x-site.readings :posts="$readings" title="برای همین مرحله" :more-href="route('blog.index')" id="cycle-readings"/>
    Redesign magazine block: heading (eyebrow «مجله») with an underlined «همه مقاله‌ها» link, then a 3-column grid of
    x-site.article-card (gap 40/24). `posts`: list<PostCardData>. Renders nothing when empty.
--}}
@props(['posts' => [], 'eyebrow' => 'مجله', 'title' => 'برای همین مرحله', 'moreHref' => null, 'moreLabel' => 'همه مقاله‌ها', 'id' => 'readings'])
@if (count($posts) > 0)
<section aria-labelledby="{{ $id }}-title" {{ $attributes->class('flex flex-col gap-10 bg-surface px-30 py-24 max-xl:px-6 max-lg:px-5 max-lg:gap-6 max-lg:py-14') }}>
    <x-site.heading :id="$id.'-title'" :eyebrow="$eyebrow" :title="$title" :more-href="$moreHref" :more-label="$moreLabel"/>
    <div class="grid grid-cols-3 gap-x-6 gap-y-10 max-lg:grid-cols-2 max-sm:grid-cols-1">
        @foreach ($posts as $post)
            <x-site.article-card :post="$post"/>
        @endforeach
    </div>
</section>
@endif
