{{--
    <x-faq :faq="$faq" eyebrow="سؤال‌های رایج" title="قبل از نصب" align="center"/>               (home)
    <x-faq :faq="$faq" variant="grid" eyebrow="قبل از خرید" title="جواب سؤال‌هایی که حق داری بپرسی" bg="surface"/>  (plus)
    <x-faq :faq="$faq" eyebrow="سؤالات متداول" title="شاید جوابت اینجا باشد" bg="surface">
        <a href="{{ route('faq') }}" class="…">همه سؤال‌ها</a>                                              (contact; slot)
    </x-faq>
    <x-faq :items="$page->faq" bare/>                                                                 (just the list)

    Reusable FAQ block (L3-09, AUDIT §2.2). Data arrives as props — never queried here:
      faq      FaqGroupData|null — a group from PageFaq::group() (also registers its FAQPage JSON-LD) or the
               `$faq` a FaqServiceProvider::PAGE_GROUPS composer hands the page view. null/empty → renders nothing.
      items    instead of `faq`: list of FaqItemData | Seo FaqItem | ['question' => …, 'answer' => …].
    Answers are printed unescaped: they are sanitised on save (FaqObserver) or trusted copy from lang files.
    The page is responsible for the JSON-LD of what it shows (PageFaq / FaqPageNode) — only visible Q&As.

      variant  list (default): native <details>, first open (`open` first|all|none), 18/800 summary, 16 lh 2 answer,
               max-w 880 · grid: 2-column white cards (plus), question 17/700 as h3, answer 15 lh 1.9, 1 column ≤ 700.
      eyebrow, title (h2), align start|center, heading-id, bg canvas|surface|none, pad xl|lg — section wrapper.
      bare     only the list/grid (no section, no header); e.g. inside /faq categories and the stage FAQ partial.
      slot     follows the list (a «همه سؤال‌ها» link).
--}}
@props([
    'faq' => null,
    'items' => null,
    'variant' => 'list',
    'eyebrow' => null,
    'title' => null,
    'align' => 'start',
    'headingId' => null,
    'bg' => 'none',
    'pad' => 'xl',
    'open' => 'first',
    'bare' => false,
])
@php
    $entries = [];
    foreach ($faq?->items ?? $items ?? [] as $item) {
        $question = is_array($item) ? ($item['question'] ?? '') : $item->question;
        $answer = is_array($item) ? ($item['answer'] ?? '') : $item->answer;
        if (trim((string) $question) !== '') {
            $entries[] = ['question' => (string) $question, 'answer' => (string) $answer];
        }
    }
    $headingId ??= 'faq-'.($faq?->slug ?? 'title');
@endphp
@if ($entries !== [] && $bare)
    <x-faq.list :entries="$entries" :variant="$variant" :open="$open" {{ $attributes }}/>
@elseif ($entries !== [])
    <x-ui.section :bg="$bg" :pad="$pad" aria-labelledby="{{ $headingId }}" {{ $attributes->class(['flex flex-col gap-8', 'items-center' => $align === 'center']) }}>
        @if ($title)
            <x-ui.section-header :eyebrow="$eyebrow" :title="$title" :align="$align" :id="$headingId"/>
        @endif
        <x-faq.list :entries="$entries" :variant="$variant" :open="$open"/>
        {{ $slot }}
    </x-ui.section>
@endif
