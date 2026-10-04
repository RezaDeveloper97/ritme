{{--
    <x-ui.toggle-row title="یادآور خرید قبل از پریود" text="فقط تاریخ تقریبی پریود بعدی"/>            (privacy: status pill)
    <x-ui.toggle-row variant="switch" title="…" text="…" :checked="false"/>                             (shop: visual switch)
    Consent row (AUDIT §2.2). `status` shows the «پیش‌فرض: خاموش» pill; `switch` shows the off switch. The site
    cannot change in-app consents, so the switch is a visual (aria-hidden) with its state spoken as text — never a fake
    focusable control. Stack several inside a card; rows after the first get a top border (`divided`).
--}}
@props(['title', 'text' => null, 'variant' => 'status', 'checked' => false, 'defaultLabel' => 'پیش‌فرض: خاموش', 'divided' => false, 'as' => 'b'])
<div {{ $attributes->class(['flex items-center justify-between gap-4 py-4.5 max-lg:flex-wrap', 'border-t border-line' => $divided]) }}>
    <div class="grow">
        <{{ $as }} class="m-0 text-lg font-bold">{{ $title }}</{{ $as }}>
        @if ($text)<div class="text-base font-semibold text-muted">{{ $text }}</div>@endif
    </div>
    @if ($variant === 'switch')
        <span class="sr-only">{{ $checked ? 'روشن' : 'خاموش' }}</span>
        <span aria-hidden="true" @class(['relative h-8 w-13 shrink-0 rounded-full', $checked ? 'bg-primary' : 'bg-line'])><span @class(['absolute top-1 size-6 rounded-full bg-surface', $checked ? 'end-1' : 'start-1'])></span></span>
    @else
        <span class="flex h-8.5 shrink-0 items-center rounded-full border border-line bg-canvas px-3.5 text-sm font-extrabold text-muted">{{ $defaultLabel }}</span>
    @endif
</div>
