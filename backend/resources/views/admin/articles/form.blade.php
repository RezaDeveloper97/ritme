@extends('admin.layouts.app')

@php $isEdit = $article->exists; @endphp
@section('title', $isEdit ? 'ویرایش مقاله' : 'مقاله جدید')

@section('content')
    <div class="page-actions">
        <a class="btn ghost" href="{{ route('admin.articles.index') }}">→ بازگشت</a>
    </div>

    <div class="card">
        <div class="card-body">
            <form method="POST" action="{{ $isEdit ? route('admin.articles.update', $article) : route('admin.articles.store') }}">
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
                    <x-admin.bilingual name="excerpt" label="خلاصه" :value="$article->excerpt" type="textarea" />
                    <x-admin.bilingual name="body" label="متن کامل" :value="$article->body" type="textarea" />

                    @include('admin.partials.phase-select', ['selected' => $article->cycle_phase, 'phases' => $phases])

                    <div class="field">
                        <label for="read_time_minutes">زمان مطالعه (دقیقه)</label>
                        <input type="number" id="read_time_minutes" name="read_time_minutes" value="{{ old('read_time_minutes', $article->read_time_minutes) }}" min="1" max="120">
                    </div>
                    <div class="field">
                        <label for="image_url">آدرس تصویر</label>
                        <input type="text" id="image_url" name="image_url" dir="ltr" value="{{ old('image_url', $article->image_url) }}">
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
