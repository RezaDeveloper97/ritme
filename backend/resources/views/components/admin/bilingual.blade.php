@props([
    'name',
    'label',
    'value' => [],
    'type' => 'text',
    'required' => false,
])

@php
    $value = is_array($value) ? $value : [];
    $fa = old(str_replace('[', '.', str_replace(']', '', $name)) . '.fa', $value['fa'] ?? '');
    $en = old(str_replace('[', '.', str_replace(']', '', $name)) . '.en', $value['en'] ?? '');
@endphp

<div class="field full">
    <label>{{ $label }}</label>
    <div class="form-grid">
        <div class="field" style="margin:0">
            <span class="hint">فارسی</span>
            @if ($type === 'textarea')
                <textarea name="{{ $name }}[fa]" dir="rtl" @if($required) required @endif>{{ $fa }}</textarea>
            @else
                <input type="text" name="{{ $name }}[fa]" dir="rtl" value="{{ $fa }}" @if($required) required @endif>
            @endif
        </div>
        <div class="field" style="margin:0">
            <span class="hint">English</span>
            @if ($type === 'textarea')
                <textarea name="{{ $name }}[en]" dir="ltr">{{ $en }}</textarea>
            @else
                <input type="text" name="{{ $name }}[en]" dir="ltr" value="{{ $en }}">
            @endif
        </div>
    </div>
</div>
