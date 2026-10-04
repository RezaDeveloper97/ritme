{{--
    Stage FAQ (AUDIT §2.4 `x-stage.faq` + §6 correction): eyebrow «سؤال‌های رایج» + a descriptive h2 («سؤال‌های رایج
    درباره …», never the bare stage name) + 3 native <details>. FAQPage JSON-LD is added by StagePageController from
    the same items. $group = FAQ group (`stage-<slug>`) — L3-09 swaps the static items for the Faq context.
--}}
@if ($items !== [])
<x-ui.section bg="surface" aria-labelledby="stage-faq-title" class="flex flex-col items-center gap-8" data-faq-group="{{ $group }}">
    <x-ui.section-header :eyebrow="$eyebrow" :title="$title" align="center" id="stage-faq-title"/>
    <x-ui.accordion :items="$items"/>
</x-ui.section>
@endif
