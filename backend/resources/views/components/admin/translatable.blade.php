@props([
    'name',
    'label',
    'value' => [],
    'type' => 'text',
    'required' => false,
])

@php
    use App\Services\Language\LanguageRegistry;

    // One input per ACTIVE language — the list comes from the languages table,
    // so adding a locale in /admin/languages grows every content form in the
    // panel with no change here. Never hardcode fa/en.
    $registry = app(LanguageRegistry::class);
    $languages = $registry->all();
    $defaultCode = $registry->defaultCode();

    $value = is_array($value) ? $value : [];
    $key = str_replace('[', '.', str_replace(']', '', $name));
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
        @foreach ($languages as $language)
            @php
                $code = $language['code'];
                $dir = $language['direction'];
                $inputId = $id . '_' . preg_replace('/[^A-Za-z0-9_]/', '_', $code);
                $current = old($key . '.' . $code, $value[$code] ?? '');
                // Only the default locale is mandatory: content must be
                // publishable before it is translated, and every reader falls
                // back to the default locale (App\Support\Translatable::pick).
                $isRequired = $required && $code === $defaultCode;
            @endphp
            <div class="field" style="margin:0">
                <span class="hint">
                    {{ $language['name'] }}
                    @if ($code === $defaultCode)
                        <span class="badge pink" style="margin-inline-start:4px">پیش‌فرض</span>
                    @endif
                </span>
                @if ($isTextarea)
                    <textarea id="{{ $inputId }}" name="{{ $name }}[{{ $code }}]" dir="{{ $dir }}"
                              @if ($isEditor) data-rich-editor data-editor-language="{{ $code }}" @endif
                              @if ($isRequired) required @endif>{{ $current }}</textarea>
                @else
                    <input type="text" id="{{ $inputId }}" name="{{ $name }}[{{ $code }}]" dir="{{ $dir }}"
                           value="{{ $current }}" @if ($isRequired) required @endif>
                @endif
                @error($key . '.' . $code) <span class="err">{{ $message }}</span> @enderror
            </div>
        @endforeach
    </div>
</div>

@if ($isEditor)
    @once
        @include('admin.partials.ckeditor')
    @endonce
@endif
