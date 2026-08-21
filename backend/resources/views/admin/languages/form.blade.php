@extends('admin.layouts.app')

@php $isEdit = $language->exists; @endphp
@section('title', $isEdit ? 'ویرایش زبان' : 'زبان جدید')

@section('content')
    <div class="page-actions">
        <a class="btn ghost" href="{{ route('admin.languages.index') }}">→ بازگشت</a>
    </div>

    <div class="card">
        <div class="card-body">
            <form method="POST" action="{{ $isEdit ? route('admin.languages.update', $language) : route('admin.languages.store') }}">
                @csrf
                @if ($isEdit) @method('PUT') @endif

                <div class="form-grid">
                    <div class="field">
                        <label for="code">کد زبان</label>
                        <input type="text" id="code" name="code" dir="ltr"
                               value="{{ old('code', $language->code) }}"
                               placeholder="ar" required @if ($isEdit) readonly @endif>
                        <span class="hint">
                            مثل <span dir="ltr">fa</span>، <span dir="ltr">en</span>، <span dir="ltr">ar</span> یا <span dir="ltr">pt-BR</span>.
                            همین کد در آدرس اپ (<span dir="ltr">/ar/home</span>) و در فایل‌های ترجمه استفاده می‌شود
                            @if ($isEdit) و پس از ساخت قابل تغییر نیست @endif.
                        </span>
                        @error('code') <span class="err">{{ $message }}</span> @enderror
                    </div>

                    <div class="field">
                        <label for="direction">جهت نوشتار</label>
                        <select id="direction" name="direction" required>
                            @foreach ($directions as $direction)
                                <option value="{{ $direction->value }}"
                                        @selected(old('direction', $language->direction?->value ?? 'ltr') === $direction->value)>
                                    {{ $direction->label('fa') }}
                                </option>
                            @endforeach
                        </select>
                        <span class="hint">اپ برای این زبان با همین جهت رندر می‌شود.</span>
                    </div>

                    <div class="field">
                        <label for="name">نام زبان (به همان زبان)</label>
                        <input type="text" id="name" name="name" value="{{ old('name', $language->name) }}"
                               placeholder="العربية" required>
                        <span class="hint">همین متن در فهرست انتخاب زبانِ اپ دیده می‌شود.</span>
                        @error('name') <span class="err">{{ $message }}</span> @enderror
                    </div>

                    <div class="field">
                        <label for="english_name">نام انگلیسی</label>
                        <input type="text" id="english_name" name="english_name" dir="ltr"
                               value="{{ old('english_name', $language->english_name) }}" placeholder="Arabic" required>
                        @error('english_name') <span class="err">{{ $message }}</span> @enderror
                    </div>

                    @unless ($isEdit)
                        <div class="field full">
                            <label for="copy_from">ساخت فایل ترجمه از روی</label>
                            <select id="copy_from" name="copy_from">
                                @foreach ($sources as $source)
                                    <option value="{{ $source->code }}" @selected($source->code === $defaultCode)>
                                        {{ $source->name }} ({{ $source->code }})
                                    </option>
                                @endforeach
                            </select>
                            <span class="hint">
                                همه‌ی رشته‌های رابط کاربری از این زبان کپی می‌شود تا اپ از همان لحظه با زبان جدید کار کند؛
                                بعد می‌توانید در صفحه‌ی «ترجمه‌ها» آن‌ها را یکی‌یکی ترجمه کنید.
                            </span>
                        </div>
                    @endunless

                    <div class="field">
                        <label for="sort_order">ترتیب نمایش</label>
                        <input type="number" id="sort_order" name="sort_order" value="{{ old('sort_order', $language->sort_order ?? 0) }}">
                    </div>

                    <div class="field" style="align-self:end">
                        <div class="switch-row">
                            <input type="checkbox" id="is_active" name="is_active" value="1" @checked(old('is_active', $language->is_active ?? true))>
                            <label for="is_active" style="margin:0">فعال</label>
                        </div>
                        <div class="switch-row" style="margin-top:8px">
                            <input type="checkbox" id="is_default" name="is_default" value="1" @checked(old('is_default', $language->is_default))>
                            <label for="is_default" style="margin:0">زبان پیش‌فرض</label>
                        </div>
                        <span class="hint">
                            زبان پیش‌فرض همیشه فعال است، در فرم‌های محتوا اجباری است و مرجع بازگشتِ رشته‌های ترجمه‌نشده است.
                        </span>
                    </div>
                </div>

                <div class="form-actions">
                    <button class="btn primary" type="submit">{{ $isEdit ? 'ذخیره تغییرات' : 'ایجاد زبان' }}</button>
                </div>
            </form>
        </div>
    </div>

    @if ($isEdit)
        <div class="card" style="margin-top:16px">
            <div class="card-body">
                <label style="display:block;margin-bottom:6px">ساخت دوباره‌ی فایل‌های ترجمه</label>
                <span class="hint">
                    رشته‌های تازه‌ای که به اپ اضافه شده‌اند را از یک زبان دیگر کپی می‌کند.
                    ترجمه‌هایی که خودتان نوشته‌اید بازنویسی می‌شوند، پس فقط وقتی استفاده کنید که می‌خواهید از نو شروع کنید.
                </span>
                <form method="POST" action="{{ route('admin.languages.regenerate', $language) }}"
                      onsubmit="return confirm('فایل‌های ترجمه‌ی این زبان دوباره ساخته شود؟')"
                      style="display:flex;gap:8px;align-items:center;margin-top:10px;flex-wrap:wrap">
                    @csrf
                    <select name="copy_from" style="max-width:220px">
                        @foreach ($sources as $source)
                            @continue($source->code === $language->code)
                            <option value="{{ $source->code }}" @selected($source->code === $defaultCode)>
                                {{ $source->name }} ({{ $source->code }})
                            </option>
                        @endforeach
                    </select>
                    <button class="btn" type="submit">ساخت دوباره</button>
                </form>
            </div>
        </div>
    @endif
@endsection
