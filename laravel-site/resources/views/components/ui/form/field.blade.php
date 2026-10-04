{{--
    <x-ui.form.field for="contact-name" label="نام" hint="…" :error="$errors->first('name')" required>
        <x-ui.form.input id="contact-name" name="name" placeholder="نام تو"/>
    </x-ui.form.field>
    Label (14/800) above the control, optional hint and error (AUDIT §2.2 forms). The control must carry the same `id`
    as `for`; pass `aria-describedby="{for}-hint {for}-error"` (x-ui.form.input does it when `described` is set).
    Use `as="fieldset"` + `legend` semantics for radio-card groups.
--}}
@props(['for' => null, 'label', 'hint' => null, 'error' => null, 'required' => false, 'as' => 'div'])
@php($legend = $as === 'fieldset')
<{{ $as }} {{ $attributes->class(['flex flex-col gap-2', 'm-0 min-w-0 border-0 p-0' => $legend]) }}>
    @if ($legend)
        <legend class="mb-2 p-0 text-base font-extrabold text-ink">{{ $label }}@if ($required)<span class="text-danger" aria-hidden="true"> *</span>@endif</legend>
    @else
        <label @if ($for) for="{{ $for }}" @endif class="text-base font-extrabold text-ink">{{ $label }}@if ($required)<span class="text-danger" aria-hidden="true"> *</span>@endif</label>
    @endif
    {{ $slot }}
    @if ($hint)<span @if ($for) id="{{ $for }}-hint" @endif class="text-sm leading-relaxed font-semibold text-muted">{{ $hint }}</span>@endif
    @if ($error)<span @if ($for) id="{{ $for }}-error" @endif class="text-sm font-bold text-danger" role="alert">{{ $error }}</span>@endif
</{{ $as }}>
