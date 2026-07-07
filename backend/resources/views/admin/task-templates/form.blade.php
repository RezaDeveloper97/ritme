@extends('admin.layouts.app')

@php $isEdit = $task->exists; @endphp
@section('title', $isEdit ? 'ویرایش کار' : 'کار جدید')

@section('content')
    <div class="page-actions"><a class="btn ghost" href="{{ route('admin.task-templates.index') }}">→ بازگشت</a></div>

    <div class="card">
        <div class="card-body">
            <form method="POST" action="{{ $isEdit ? route('admin.task-templates.update', $task) : route('admin.task-templates.store') }}">
                @csrf
                @if ($isEdit) @method('PUT') @endif

                <div class="form-grid">
                    <div class="field">
                        <label for="key">شناسه یکتا (key)</label>
                        <input type="text" id="key" name="key" dir="ltr" value="{{ old('key', $task->key) }}" required>
                        @error('key') <span class="err">{{ $message }}</span> @enderror
                    </div>
                    <div class="field">
                        <label for="category">دسته</label>
                        <select id="category" name="category" required>
                            @foreach ($categories as $c)
                                <option value="{{ $c->value }}" @selected(old('category', $task->category) === $c->value)>{{ $c->label('fa') }}</option>
                            @endforeach
                        </select>
                    </div>

                    <x-admin.bilingual name="title" label="عنوان" :value="$task->title" required />
                    <x-admin.bilingual name="description" label="توضیحات" :value="$task->description" type="textarea" />

                    @include('admin.partials.phase-select', ['selected' => $task->cycle_phase, 'phases' => $phases])
                    <div class="field">
                        <label for="icon">آیکون</label>
                        <input type="text" id="icon" name="icon" dir="ltr" value="{{ old('icon', $task->icon) }}">
                    </div>
                    <div class="field">
                        <label for="sort_order">ترتیب نمایش</label>
                        <input type="number" id="sort_order" name="sort_order" value="{{ old('sort_order', $task->sort_order ?? 0) }}">
                    </div>
                    <div class="field switch-row" style="align-self:end">
                        <input type="checkbox" id="is_active" name="is_active" value="1" @checked(old('is_active', $task->is_active))>
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
