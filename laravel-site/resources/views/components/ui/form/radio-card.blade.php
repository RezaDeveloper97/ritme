{{--
    <x-ui.form.radio-card name="pay" value="cod" icon="credit-card" title="پرداخت در محل" text="…" :checked="true"/>
    Selectable card (payment, delivery slot, booking day/time, size — AUDIT §2.2): radius 22, 1.5px border; the checked
    state (primary border + primary/11 fill) follows the real radio via CSS `has-checked:` — no JS. Group several in
    <x-ui.form.field as="fieldset" label="روش پرداخت">. `type="checkbox"` for multi-select.
--}}
@props(['name', 'value', 'title', 'text' => null, 'icon' => null, 'checked' => false, 'disabled' => false, 'type' => 'radio'])
<label {{ $attributes->class('flex min-h-11 cursor-pointer items-center gap-3 rounded-[22px] border-[1.5px] border-line bg-surface px-4 py-3.5 transition-colors has-checked:border-primary has-checked:bg-primary/11 has-disabled:cursor-not-allowed has-disabled:opacity-50 has-focus-visible:outline-2 has-focus-visible:outline-primary max-lg:flex-wrap') }}>
    @if ($icon)<span aria-hidden="true" class="flex size-10.5 shrink-0 items-center justify-center rounded-full bg-primary/13 text-primary"><x-icon :name="$icon" class="size-[21px]"/></span>@endif
    <span class="grow">
        <b class="text-base">{{ $title }}</b>
        @if ($text)<span class="block text-xs leading-[1.7] font-semibold text-muted">{{ $text }}</span>@endif
    </span>
    <input type="{{ $type }}" name="{{ $name }}" value="{{ $value }}" @checked($checked) @disabled($disabled) class="size-5.5 shrink-0 accent-primary">
</label>
