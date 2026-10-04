{{--
    <x-ui.form.input id="contact-email" name="email" type="email" placeholder="برای پاسخ" :invalid="$errors->has('email')" described/>
    Text input: 56px, radius 18, 1.5px line border, 15/600 (AUDIT §2.2). `described` links `{id}-hint {id}-error`
    from x-ui.form.field; `invalid` sets aria-invalid + the danger border. Old input is the caller's job (`:value="old('email')"`).
--}}
@props(['type' => 'text', 'invalid' => false, 'described' => false])
<input type="{{ $type }}" @if ($invalid) aria-invalid="true" @endif @if ($described && $attributes->has('id')) aria-describedby="{{ $attributes->get('id') }}-hint {{ $attributes->get('id') }}-error" @endif
    {{ $attributes->class(['min-h-14 w-full rounded-2xl border-[1.5px] bg-surface px-4.5 text-md font-semibold text-ink placeholder:text-muted', $invalid ? 'border-danger' : 'border-line focus:border-primary']) }}>
