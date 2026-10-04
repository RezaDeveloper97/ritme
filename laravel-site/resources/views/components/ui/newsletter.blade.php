{{--
    <x-ui.newsletter title="هر هفته یک خواندنی کوتاه" text="مخصوص مرحله خودت؛ بدون تبلیغ." :action="null"/>
    Lavender subscribe box (AUDIT §2.2, blog). A real labelled form (the design's input is a <span>): text input
    «ایمیل یا شماره همراه» + «عضویت». L4-02 wires it by passing `action` (POST + @csrf). Without an action the field
    and button render disabled so nothing pretends to work. Slot: extra hidden inputs / consent text.
--}}
@props([
    'title' => 'هر هفته یک خواندنی کوتاه',
    'text' => 'مخصوص مرحله خودت؛ بدون تبلیغ.',
    'action' => null,
    'placeholder' => 'ایمیل یا شماره همراه',
    'button' => 'عضویت',
    'name' => 'contact',
    'id' => 'newsletter',
])
<section aria-labelledby="{{ $id }}-title" {{ $attributes->class('flex items-center gap-8 rounded-6xl bg-lavender p-9 max-lg:flex-wrap max-sm:p-6') }}>
    <div class="grow">
        <h2 id="{{ $id }}-title" class="m-0 font-display text-[30px] leading-heading font-normal text-ink">{{ $title }}</h2>
        @if ($text)<p class="m-0 text-lg leading-loose font-medium text-muted">{{ $text }}</p>@endif
    </div>
    <form @if ($action) action="{{ $action }}" method="post" @endif class="flex gap-2.5 max-lg:flex-wrap max-sm:w-full">
        @if ($action) @csrf @endif
        <label for="{{ $id }}-input" class="sr-only">{{ $placeholder }}</label>
        <input id="{{ $id }}-input" name="{{ $name }}" type="text" inputmode="email" autocomplete="email" required placeholder="{{ $placeholder }}" @disabled(! $action)
               class="h-13.5 w-85 rounded-full border-[1.5px] border-line bg-surface px-5 text-md text-ink placeholder:text-muted focus:border-primary disabled:cursor-not-allowed max-sm:w-full">
        <button type="submit" @disabled(! $action) class="inline-flex h-13.5 items-center gap-2 rounded-full border-[1.5px] border-primary bg-primary px-6.5 text-[15.5px] font-extrabold text-white transition-colors hover:bg-primary-hover disabled:cursor-not-allowed">{{ $button }}</button>
        {{ $slot }}
    </form>
</section>
