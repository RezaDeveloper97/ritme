{{--
    <x-site.faq :items="[['question' => …, 'answer' => …], …]" title="سؤالات متداول" id="cycle-faq"/>
    Redesign FAQ: centred Lalezar 40 h2, then a single 880px column of <details> rows (18/800 question, plus icon
    that turns into a ×, 16 lh 2 muted answer); the first row starts open. Answers may hold the admin's sanitised
    HTML (FaqItem answers are cleaned on save), so they are echoed raw — pass `escape` for plain-text sources.
    Nothing renders without items. The FAQPage JSON-LD is registered by the page, not here.
--}}
@props(['items' => [], 'title' => 'سؤالات متداول', 'id' => 'faq', 'escape' => false, 'eyebrow' => null])
@if (count($items) > 0)
<section aria-labelledby="{{ $id }}-title" {{ $attributes->class('flex flex-col gap-10 bg-surface px-30 pb-24 max-xl:px-6 max-lg:px-5 max-lg:gap-6 max-lg:pb-14') }}>
    <x-site.heading :id="$id.'-title'" :title="$title" :eyebrow="$eyebrow" align="center"/>
    <div class="mx-auto flex w-full max-w-220 flex-col border-t border-line">
        @foreach ($items as $item)
            <details class="group border-b border-line py-5" @if ($loop->first) open @endif>
                <summary class="flex min-h-8 items-center justify-between gap-4 text-2xl font-extrabold text-ink max-sm:text-lg">
                    {{ $item['question'] }}
                    <x-icon name="plus" class="size-4.5 shrink-0 text-muted transition-transform group-open:rotate-45 motion-reduce:transition-none"/>
                </summary>
                <div class="mt-2.5 text-lg leading-loose text-muted [&_a]:font-extrabold [&_a]:text-primary [&_p+p]:mt-3 [&_ul]:list-disc [&_ul]:ps-5">{!! $escape ? e($item['answer']) : $item['answer'] !!}</div>
            </details>
        @endforeach
    </div>
</section>
@endif
