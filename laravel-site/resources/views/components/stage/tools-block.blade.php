{{--
    <x-stage.tools-block :items="[
        ['href' => route('tools'), 'icon' => 'calculator', 'color' => 'ttc', 'title' => 'محاسبه روزهای باروری', 'text' => '…', 'where' => 'روی سایت'],
        …
    ]"/>
    «کارهای کوچک، آمادگی بیشتر» (AUDIT §2.4, 6 stage pages): eyebrow «ابزارهای این مرحله» + h2 + a 3-column grid of
    x-cards.feature. Heading copy is overridable; items are feature-card props.
--}}
@props(['items' => [], 'eyebrow' => 'ابزارهای این مرحله', 'title' => 'کارهای کوچک، آمادگی بیشتر', 'id' => 'stage-tools', 'bg' => 'none'])
<x-ui.section :bg="$bg" aria-labelledby="{{ $id }}-title" {{ $attributes->class('flex flex-col gap-8') }}>
    <x-ui.section-header :eyebrow="$eyebrow" :title="$title" :id="$id.'-title'"/>
    <div class="grid grid-cols-3 gap-4 max-sm:grid-cols-1">
        @foreach ($items as $item)
            <x-cards.feature :href="$item['href'] ?? null" :icon="$item['icon']" :color="$item['color'] ?? 'primary'" :title="$item['title']" :text="$item['text'] ?? null" :where="$item['where'] ?? null"/>
        @endforeach
    </div>
</x-ui.section>
