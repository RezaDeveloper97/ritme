@extends('admin.layouts.app')

@php $isEdit = $recommendation->exists; @endphp
@section('title', $isEdit ? 'ویرایش توصیه' : 'توصیه جدید')

@section('content')
    <div class="page-actions"><a class="btn ghost" href="{{ route('admin.recommendations.index') }}">→ بازگشت</a></div>

    <div class="card">
        <div class="card-body">
            <form method="POST" action="{{ $isEdit ? route('admin.recommendations.update', $recommendation) : route('admin.recommendations.store') }}">
                @csrf
                @if ($isEdit) @method('PUT') @endif

                <div class="form-grid">
                    <div class="field">
                        <label for="type">دسته</label>
                        <select id="type" name="type" required>
                            @foreach ($types as $t)
                                <option value="{{ $t['value'] }}" @selected(old('type', $recommendation->type) === $t['value'])>{{ $t['label'] }}</option>
                            @endforeach
                        </select>
                        <span class="hint">آیکون و عنوان پیش‌فرض کارت از روی دسته انتخاب می‌شود.</span>
                        @error('type') <span class="err">{{ $message }}</span> @enderror
                    </div>

                    @include('admin.partials.phase-select', ['selected' => $recommendation->cycle_phase, 'phases' => $phases])

                    <x-admin.bilingual name="text" label="متن توصیه" :value="$recommendation->text" type="textarea" required />

                    <x-admin.bilingual name="title" label="عنوان (اختیاری)" :value="$recommendation->title" />
                    <div class="field full">
                        <span class="hint">اگر خالی بماند، عنوان دسته (مثلاً «تغذیه») نمایش داده می‌شود.</span>
                    </div>

                    @include('admin.recommendations.subphase-picker', [
                        'subphases' => $subphases,
                        'phaseOf' => $subphasePhases,
                        'selected' => $recommendation->cycle_subphases ?? [],
                    ])

                    <div class="field">
                        <label for="symptom_trigger">وابسته به علامت (اختیاری)</label>
                        <select id="symptom_trigger" name="symptom_trigger">
                            <option value="">— بدون شرط —</option>
                            @foreach ($triggers as $trigger)
                                <option value="{{ $trigger['value'] }}" @selected(old('symptom_trigger', $recommendation->symptom_trigger) === $trigger['value'])>{{ $trigger['label'] }}</option>
                            @endforeach
                        </select>
                        <span class="hint">فقط روزهایی نمایش داده می‌شود که کاربر این علامت را ثبت کرده باشد.</span>
                        @error('symptom_trigger') <span class="err">{{ $message }}</span> @enderror
                    </div>

                    <div class="field">
                        <label for="sort_order">ترتیب نمایش</label>
                        <input type="number" id="sort_order" name="sort_order" value="{{ old('sort_order', $recommendation->sort_order ?? 0) }}">
                        <span class="hint">عدد کوچک‌تر بالاتر نمایش داده می‌شود.</span>
                    </div>

                    <div class="field switch-row" style="align-self:end">
                        <input type="checkbox" id="is_active" name="is_active" value="1" @checked(old('is_active', $recommendation->is_active))>
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
