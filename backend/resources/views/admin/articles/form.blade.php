@extends('admin.layouts.app')

@php $isEdit = $article->exists; @endphp
@section('title', $isEdit ? 'ویرایش مقاله' : 'مقاله جدید')

@section('content')
    <div class="page-actions">
        <a class="btn ghost" href="{{ route('admin.articles.index') }}">→ بازگشت</a>
    </div>

    <div class="card">
        <div class="card-body">
            <form method="POST" enctype="multipart/form-data"
                  action="{{ $isEdit ? route('admin.articles.update', $article) : route('admin.articles.store') }}">
                @csrf
                @if ($isEdit) @method('PUT') @endif

                <div class="form-grid">
                    <div class="field">
                        <label for="slug">اسلاگ (شناسه یکتا)</label>
                        <input type="text" id="slug" name="slug" dir="ltr" value="{{ old('slug', $article->slug) }}" required>
                        @error('slug') <span class="err">{{ $message }}</span> @enderror
                    </div>
                    <div class="field">
                        <label for="category">دسته</label>
                        <input type="text" id="category" name="category" value="{{ old('category', $article->category) }}">
                    </div>

                    <x-admin.bilingual name="title" label="عنوان" :value="$article->title" required />
                    <x-admin.bilingual name="excerpt" label="خلاصه" :value="$article->excerpt" type="editor" />
                    <x-admin.bilingual name="body" label="متن کامل" :value="$article->body" type="editor" />

                    @include('admin.partials.phase-multi-select', ['selected' => $article->cycle_phases ?? [], 'phases' => $phases])

                    <div class="field">
                        <label for="read_time_minutes">زمان مطالعه (دقیقه)</label>
                        <input type="number" id="read_time_minutes" name="read_time_minutes" value="{{ old('read_time_minutes', $article->read_time_minutes) }}" min="1" max="120">
                    </div>

                    <div class="field full">
                        <label for="image">تصویر مقاله</label>
                        <span class="hint">
                            فقط فایل تصویری (JPG، PNG یا WebP)، حداکثر ۸ مگابایت.
                            تصویر پس از آپلود به‌صورت خودکار بهینه و حداکثر تا {{ $maxImageWidth }} پیکسل کوچک می‌شود.
                        </span>
                        @if ($article->image_url)
                            <img src="{{ $article->image_url }}" alt=""
                                 style="width:100%;max-width:320px;aspect-ratio:16/9;object-fit:cover;border-radius:12px;border:1px solid var(--border);margin:6px 0">
                        @endif
                        <input type="file" id="image" name="image" accept="image/jpeg,image/png,image/webp">
                        @error('image') <span class="err">{{ $message }}</span> @enderror

                        @if ($article->image_path)
                            <div class="switch-row" style="margin-top:6px">
                                <input type="checkbox" id="remove_image" name="remove_image" value="1">
                                <label for="remove_image" style="margin:0;font-weight:500">حذف تصویر فعلی</label>
                            </div>
                        @endif
                    </div>

                    <div class="field full">
                        <label for="image_url">یا آدرس تصویر (لینک خارجی — اختیاری)</label>
                        <input type="text" id="image_url" name="image_url" dir="ltr" value="{{ old('image_url', $article->getRawOriginal('image_url')) }}">
                        <span class="hint">اگر فایلی آپلود شود، همان نمایش داده می‌شود و این آدرس نادیده گرفته می‌شود.</span>
                        @error('image_url') <span class="err">{{ $message }}</span> @enderror
                    </div>

                    <div class="field">
                        <label for="sort_order">ترتیب نمایش</label>
                        <input type="number" id="sort_order" name="sort_order" value="{{ old('sort_order', $article->sort_order ?? 0) }}">
                    </div>
                    <div class="field switch-row" style="align-self:end">
                        <input type="checkbox" id="is_published" name="is_published" value="1" @checked(old('is_published', $article->is_published))>
                        <label for="is_published" style="margin:0">منتشر شود</label>
                    </div>
                </div>

                <div class="form-actions">
                    <button class="btn primary" type="submit">{{ $isEdit ? 'ذخیره تغییرات' : 'ایجاد مقاله' }}</button>
                </div>
            </form>
        </div>
    </div>
@endsection
