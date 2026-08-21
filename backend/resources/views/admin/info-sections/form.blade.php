@extends('admin.layouts.app')

@php
    $isEdit = $section->exists;
    $groupLabel = \App\Models\InfoSection::groupLabel($group);
@endphp
@section('title', ($isEdit ? 'ویرایش باکس' : 'باکس جدید') . ' — ' . $groupLabel)

@section('content')
    <div class="page-actions">
        <a class="btn ghost" href="{{ route('admin.info-sections.index', ['group' => $group]) }}">→ بازگشت به {{ $groupLabel }}</a>
    </div>

    <div class="card">
        <div class="card-body">
            <form method="POST" action="{{ $isEdit ? route('admin.info-sections.update', $section) : route('admin.info-sections.store') }}">
                @csrf
                @if ($isEdit) @method('PUT') @endif

                <div class="form-grid">
                    <div class="field full">
                        <label for="group">صفحه</label>
                        <select id="group" name="group">
                            @foreach (\App\Models\InfoSection::GROUPS as $option)
                                <option value="{{ $option }}" @selected(old('group', $group) === $option)>
                                    {{ \App\Models\InfoSection::groupLabel($option) }}
                                </option>
                            @endforeach
                        </select>
                        <span class="hint">باکس در همین صفحه از اپ نمایش داده می‌شود.</span>
                    </div>

                    <x-admin.translatable name="heading" label="عنوان باکس" :value="$section->heading" required />
                    <x-admin.translatable name="body" label="متن باکس" :value="$section->body" type="textarea" required />

                    <div class="field full">
                        <label style="margin-bottom:2px">دکمه تماس (اختیاری)</label>
                        <span class="hint">
                            اگر می‌خواهید زیر متن یک دکمه باشد، هم برچسب و هم آدرس را پر کنید؛
                            در غیر این صورت هر دو را خالی بگذارید. مثال آدرس:
                            <span dir="ltr">mailto:support@ritmesalamat.com</span>،
                            <span dir="ltr">tel:+982112345678</span>،
                            <span dir="ltr">https://t.me/ritme</span>
                        </span>
                    </div>
                    <x-admin.translatable name="link_label" label="برچسب دکمه" :value="$section->link_label" />
                    <div class="field full">
                        <label for="link_url">آدرس دکمه</label>
                        <input type="text" id="link_url" name="link_url" dir="ltr"
                               value="{{ old('link_url', $section->link_url) }}"
                               placeholder="mailto:support@ritmesalamat.com">
                    </div>

                    <div class="field">
                        <label for="sort_order">ترتیب نمایش</label>
                        <input type="number" id="sort_order" name="sort_order" value="{{ old('sort_order', $section->sort_order ?? 0) }}">
                    </div>
                    <div class="field switch-row" style="align-self:end">
                        <input type="checkbox" id="is_active" name="is_active" value="1" @checked(old('is_active', $section->is_active))>
                        <label for="is_active" style="margin:0">فعال</label>
                    </div>
                </div>

                <div class="form-actions">
                    <button class="btn primary" type="submit">{{ $isEdit ? 'ذخیره تغییرات' : 'ایجاد' }}</button>
                </div>
            </form>
        </div>
    </div>
@endsection
