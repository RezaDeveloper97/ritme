{{--
    <x-ui.form.select id="join-city" name="city" :options="['tehran' => 'تهران', 'karaj' => 'کرج']" :selected="old('city')" placeholder="انتخاب کن"/>
    Native <select> styled like x-ui.form.input with a chevron (chevron-left rotated, pointing down) at the end.
--}}
@props(['options' => [], 'selected' => null, 'placeholder' => null, 'invalid' => false])
<span class="relative block">
    <select @if ($invalid) aria-invalid="true" @endif {{ $attributes->class(['min-h-14 w-full appearance-none rounded-2xl border-[1.5px] bg-surface ps-4.5 pe-11 text-md font-semibold text-ink', $invalid ? 'border-danger' : 'border-line focus:border-primary']) }}>
        @if ($placeholder !== null)<option value="" @selected($selected === null || $selected === '')>{{ $placeholder }}</option>@endif
        @foreach ($options as $value => $label)
            <option value="{{ $value }}" @selected((string) $selected === (string) $value)>{{ $label }}</option>
        @endforeach
    </select>
    <x-icon name="chevron-left" class="pointer-events-none absolute end-4 top-1/2 size-4 -translate-y-1/2 -rotate-90 text-muted"/>
</span>
