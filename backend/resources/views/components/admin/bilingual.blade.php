@props([
    'name',
    'label',
    'value' => [],
    'type' => 'text',
    'required' => false,
])

@php
    $value = is_array($value) ? $value : [];
    $key = str_replace('[', '.', str_replace(']', '', $name));
    $fa = old($key . '.fa', $value['fa'] ?? '');
    $en = old($key . '.en', $value['en'] ?? '');
    // "editor" renders a textarea that admin.partials.ckeditor upgrades to a
    // rich-text editor; it degrades to the plain textarea if the script fails.
    // Don't combine it with required: CKEditor hides the textarea, and the
    // browser can't focus a hidden control to report the error — validate the
    // field server-side instead.
    $isEditor = $type === 'editor';
    $isTextarea = $isEditor || $type === 'textarea';
    $id = preg_replace('/[^A-Za-z0-9_]/', '_', $key);
@endphp

<div class="field full">
    <label>{{ $label }}</label>
    <div class="form-grid">
        <div class="field" style="margin:0">
            <span class="hint">فارسی</span>
            @if ($isTextarea)
                <textarea id="{{ $id }}_fa" name="{{ $name }}[fa]" dir="rtl"
                          @if ($isEditor) data-rich-editor data-editor-language="fa" @endif
                          @if ($required) required @endif>{{ $fa }}</textarea>
            @else
                <input type="text" name="{{ $name }}[fa]" dir="rtl" value="{{ $fa }}" @if($required) required @endif>
            @endif
        </div>
        <div class="field" style="margin:0">
            <span class="hint">English</span>
            @if ($isTextarea)
                <textarea id="{{ $id }}_en" name="{{ $name }}[en]" dir="ltr"
                          @if ($isEditor) data-rich-editor data-editor-language="en" @endif>{{ $en }}</textarea>
            @else
                <input type="text" name="{{ $name }}[en]" dir="ltr" value="{{ $en }}">
            @endif
        </div>
    </div>
</div>

@if ($isEditor)
    @once
        @include('admin.partials.ckeditor')
    @endonce
@endif
