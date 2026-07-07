@extends('admin.layouts.app')

@php
    $isEdit = $banner->exists;
    $dt = fn ($v) => $v ? \Illuminate\Support\Carbon::parse($v)->format('Y-m-d\TH:i') : '';
@endphp
@section('title', $isEdit ? 'ویرایش بنر' : 'بنر جدید')

@section('content')
    <div class="page-actions">
        <a class="btn ghost" href="{{ route('admin.banners.index') }}">→ بازگشت</a>
    </div>

    <div class="card">
        <div class="card-body">
            <form method="POST" enctype="multipart/form-data"
                  action="{{ $isEdit ? route('admin.banners.update', $banner) : route('admin.banners.store') }}">
                @csrf
                @if ($isEdit) @method('PUT') @endif

                <div class="form-grid">
                    <div class="field full">
                        <label for="image">تصویر بنر</label>
                        <span class="hint">اندازه پیشنهادی: {{ $recommended }} پیکسل (نسبت ۲:۱). فرمت JPG/PNG/WebP، حداکثر ۴ مگابایت.</span>
                        @if ($banner->image_url)
                            <img src="{{ $banner->image_url }}" alt=""
                                 style="width:100%;max-width:360px;aspect-ratio:2/1;object-fit:cover;border-radius:12px;border:1px solid var(--line);margin:6px 0">
                        @endif
                        <input type="file" id="image" name="image" accept="image/jpeg,image/png,image/webp" @if(! $isEdit) required @endif>
                        @error('image') <span class="err">{{ $message }}</span> @enderror
                    </div>

                    <div class="field">
                        <label for="position">جایگاه نمایش</label>
                        <select id="position" name="position" required>
                            @foreach ($positions as $pos)
                                <option value="{{ $pos->value }}" @selected(old('position', $banner->position) === $pos->value)>
                                    {{ $pos->label('fa') }}
                                </option>
                            @endforeach
                        </select>
                        @error('position') <span class="err">{{ $message }}</span> @enderror
                    </div>
                    <div class="field">
                        <label for="sort_order">ترتیب نمایش (کوچک‌تر = اول)</label>
                        <input type="number" id="sort_order" name="sort_order" value="{{ old('sort_order', $banner->sort_order ?? 0) }}">
                    </div>

                    <x-admin.bilingual name="title" label="عنوان / متن جایگزین (اختیاری)" :value="$banner->title" />

                    <div class="field">
                        <label for="link_type">نوع لینک</label>
                        <select id="link_type" name="link_type">
                            <option value="">بدون لینک</option>
                            @foreach ($linkTypes as $lt)
                                <option value="{{ $lt->value }}" @selected(old('link_type', $banner->link_type) === $lt->value)>
                                    {{ $lt->label('fa') }}
                                </option>
                            @endforeach
                        </select>
                        @error('link_type') <span class="err">{{ $message }}</span> @enderror
                    </div>
                    <div class="field">
                        <label for="link_url">آدرس لینک</label>
                        <input type="text" id="link_url" name="link_url" dir="ltr"
                               placeholder="داخلی: /calendar — خارجی: https://example.com"
                               value="{{ old('link_url', $banner->link_url) }}">
                        @error('link_url') <span class="err">{{ $message }}</span> @enderror
                    </div>

                    <div class="field">
                        <label for="starts_at">شروع نمایش (اختیاری)</label>
                        <input type="datetime-local" id="starts_at" name="starts_at" value="{{ old('starts_at', $dt($banner->starts_at)) }}">
                        @error('starts_at') <span class="err">{{ $message }}</span> @enderror
                    </div>
                    <div class="field">
                        <label for="ends_at">پایان نمایش (اختیاری)</label>
                        <input type="datetime-local" id="ends_at" name="ends_at" value="{{ old('ends_at', $dt($banner->ends_at)) }}">
                        @error('ends_at') <span class="err">{{ $message }}</span> @enderror
                    </div>

                    <div class="field switch-row" style="align-self:end">
                        <input type="checkbox" id="is_active" name="is_active" value="1" @checked(old('is_active', $banner->is_active ?? true))>
                        <label for="is_active" style="margin:0">فعال باشد</label>
                    </div>
                </div>

                <div class="form-actions">
                    <button class="btn primary" type="submit">{{ $isEdit ? 'ذخیره تغییرات' : 'ایجاد بنر' }}</button>
                </div>
            </form>
        </div>
    </div>
@endsection
