{{--
    <x-ui.newsletter title="هر هفته یک خواندنی کوتاه" text="مخصوص مرحله خودت؛ بدون تبلیغ." :action="null"/>
    <x-ui.newsletter :action="route('newsletter.store')" name="email" :value="old('email')" describedby="nl-status">
        <x-slot:error>{{ $errors->newsletter->first('email') }}</x-slot:error>
    </x-ui.newsletter>
    Lavender subscribe box (AUDIT §2.2, blog). A real labelled form (the design's input is a <span>): text input
    «ایمیل یا شماره همراه» + «عضویت». L4-02 wires it by passing `action` (POST + @csrf). Without an action the field
    and button render disabled so nothing pretends to work. `value` refills the field (old input); `describedby` adds
    ids to the input's aria-describedby. Named slot `error` (rendered only when non-empty): a full-width alert row
    inside the box (`#{id}-error`), the input gets aria-invalid + aria-describedby. Default slot: extra hidden inputs.
    Padding stays 36px on mobile like the design; the field goes full width there instead of overflowing.
--}}
@props([
    'title' => 'هر هفته یک خواندنی کوتاه',
    'text' => 'مخصوص مرحله خودت؛ بدون تبلیغ.',
    'action' => null,
    'placeholder' => 'ایمیل یا شماره همراه',
    'button' => 'عضویت',
    'name' => 'contact',
    'id' => 'newsletter',
    'value' => null,
    'describedby' => null,
    'error' => null,
])
@php
    $hasError = $error !== null && trim((string) $error) !== '';
    $describedIds = trim(implode(' ', array_filter([(string) $describedby, $hasError ? $id.'-error' : ''])));
@endphp
<section aria-labelledby="{{ $id }}-title" {{ $attributes->class(['flex items-center gap-8 rounded-6xl bg-lavender p-9 max-lg:flex-wrap', 'flex-wrap' => $hasError]) }}>
    <div class="grow">
        <h2 id="{{ $id }}-title" class="m-0 font-display text-[30px] leading-heading font-normal text-ink">{{ $title }}</h2>
        @if ($text)<p class="m-0 text-lg leading-loose font-medium text-muted">{{ $text }}</p>@endif
    </div>
    <form @if ($action) action="{{ $action }}" method="post" @endif class="flex gap-2.5 max-lg:flex-wrap max-sm:w-full">
        @if ($action) @csrf @endif
        <label for="{{ $id }}-input" class="sr-only">{{ $placeholder }}</label>
        <input id="{{ $id }}-input" name="{{ $name }}" type="text" inputmode="email" autocomplete="email" required placeholder="{{ $placeholder }}" @disabled(! $action)
               @if ($value !== null && $value !== '') value="{{ $value }}" @endif
               @if ($describedIds !== '') aria-describedby="{{ $describedIds }}" @endif
               @if ($hasError) aria-invalid="true" @endif
               class="h-13.5 w-85 rounded-full border-[1.5px] border-line bg-surface px-5 text-md text-ink placeholder:text-muted focus:border-primary disabled:cursor-not-allowed aria-invalid:border-danger max-sm:w-full">
        <button type="submit" @disabled(! $action) class="inline-flex h-13.5 items-center gap-2 rounded-full border-[1.5px] border-primary bg-primary px-6.5 text-[15.5px] font-extrabold text-white transition-colors hover:bg-primary-hover disabled:cursor-not-allowed">{{ $button }}</button>
        {{ $slot }}
    </form>
    @if ($hasError)
        <p id="{{ $id }}-error" role="alert" class="m-0 flex basis-full items-center gap-2 rounded-3xl bg-danger-soft px-5 py-3.5 text-base font-bold text-danger">
            <x-icon name="alert-triangle" class="size-4.5 shrink-0"/>{{ $error }}
        </p>
    @endif
</section>
