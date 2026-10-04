{{--
    <x-stage.readings :posts="$readings" more-href="{{ route('blog.index') }}"/>
    «برای همین مرحله» (AUDIT §2.4, 6 stage pages; index «خواندنی‌های این هفته» with more-label «همه مقاله‌ها»): eyebrow
    «مجله» + h2 + trailing «همه» link + 3 x-cards.article. `posts`: list<PostCardData> (preferred) or arrays of
    article-card props. Renders nothing when empty (a stage without posts shows no empty shell).
--}}
@props(['posts' => [], 'eyebrow' => 'مجله', 'title' => 'برای همین مرحله', 'moreHref' => null, 'moreLabel' => 'همه', 'id' => 'stage-readings', 'bg' => 'none'])
@if (count($posts) > 0)
<x-ui.section :bg="$bg" aria-labelledby="{{ $id }}-title" {{ $attributes->class('flex flex-col gap-8') }}>
    <x-ui.section-header :eyebrow="$eyebrow" :title="$title" :id="$id.'-title'" :more-href="$moreHref" :more-label="$moreLabel"/>
    <div class="grid grid-cols-3 gap-4.5 max-sm:grid-cols-1">
        @foreach ($posts as $post)
            @if ($post instanceof \App\Domain\Blog\Data\PostCardData)
                <x-cards.article :post="$post"/>
            @else
                <x-cards.article :href="$post['href']" :title="$post['title']" :stage="$post['stage'] ?? null" :label="$post['label'] ?? null" :minutes="$post['minutes'] ?? null" :media="$post['media'] ?? null"/>
            @endif
        @endforeach
    </div>
</x-ui.section>
@endif
