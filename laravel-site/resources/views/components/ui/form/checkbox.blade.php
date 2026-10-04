{{--
    <x-ui.form.checkbox name="terms" value="1" :checked="old('terms')">شرایط استفاده را خواندم</x-ui.form.checkbox>
    Native checkbox (22px, primary accent) with its label text as the slot.
--}}
@props(['checked' => false])
<label class="flex cursor-pointer items-start gap-2.5 text-md leading-relaxed font-semibold text-ink">
    <input type="checkbox" @checked($checked) {{ $attributes->class('mt-1 size-5.5 shrink-0 accent-primary') }}>
    <span>{{ $slot }}</span>
</label>
