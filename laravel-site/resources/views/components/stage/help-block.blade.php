{{--
    <x-stage.help-block :items="[
        ['href' => route('services'), 'icon' => 'stethoscope', 'color' => 'postpartum', 'title' => 'پزشک و ماما', 'text' => '…'],
        …
    ]"/>
    «وقتی کمک بیشتری لازم داری» (AUDIT §2.4: cycle, ttc, pregnancy, postpartum, menopause; index titles it «وقتی به کمک
    بیشتری نیاز داری» and drops the eyebrow → pass `title` / `:eyebrow="null"` / `bg="none"`). White band, eyebrow
    «خدمات مرتبط», 3-column x-cards.service.
--}}
@props(['items' => [], 'eyebrow' => 'خدمات مرتبط', 'title' => 'وقتی کمک بیشتری لازم داری', 'id' => 'stage-help', 'bg' => 'surface'])
<x-ui.section :bg="$bg" aria-labelledby="{{ $id }}-title" {{ $attributes->class('flex flex-col gap-8') }}>
    <x-ui.section-header :eyebrow="$eyebrow" :title="$title" :id="$id.'-title'"/>
    <div class="grid grid-cols-3 gap-4.5 max-sm:grid-cols-1">
        @foreach ($items as $item)
            <x-cards.service :href="$item['href']" :icon="$item['icon']" :color="$item['color'] ?? 'primary'" :title="$item['title']" :text="$item['text'] ?? null" :cta="$item['cta'] ?? 'بیشتر'"/>
        @endforeach
    </div>
</x-ui.section>
