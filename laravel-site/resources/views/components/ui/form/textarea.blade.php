{{--
    <x-ui.form.textarea id="contact-message" name="message" rows="5" placeholder="بنویس…" described>{{ old('message') }}</x-ui.form.textarea>
    Multi-line field: min 140px, radius 18, same border/typography as x-ui.form.input.
--}}
@props(['invalid' => false, 'described' => false])
<textarea @if ($invalid) aria-invalid="true" @endif @if ($described && $attributes->has('id')) aria-describedby="{{ $attributes->get('id') }}-hint {{ $attributes->get('id') }}-error" @endif
    {{ $attributes->merge(['rows' => 5])->class(['min-h-35 w-full rounded-2xl border-[1.5px] bg-surface px-4.5 py-4 text-md leading-relaxed font-semibold text-ink placeholder:text-muted', $invalid ? 'border-danger' : 'border-line focus:border-primary']) }}>{{ $slot }}</textarea>
