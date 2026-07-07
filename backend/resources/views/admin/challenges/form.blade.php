@extends('admin.layouts.app')

@php $isEdit = $challenge->exists; @endphp
@section('title', $isEdit ? 'ویرایش چالش' : 'چالش جدید')

@section('content')
    <div class="page-actions"><a class="btn ghost" href="{{ route('admin.challenges.index') }}">→ بازگشت</a></div>

    <div class="card">
        <div class="card-body">
            <form method="POST" action="{{ $isEdit ? route('admin.challenges.update', $challenge) : route('admin.challenges.store') }}">
                @csrf
                @if ($isEdit) @method('PUT') @endif

                <div class="form-grid">
                    <x-admin.bilingual name="title" label="عنوان" :value="$challenge->title" required />
                    <x-admin.bilingual name="description" label="توضیحات" :value="$challenge->description" type="textarea" />
                    @include('admin.partials.phase-select', ['selected' => $challenge->cycle_phase, 'phases' => $phases])
                    <div class="field">
                        <label for="category">دسته</label>
                        <input type="text" id="category" name="category" value="{{ old('category', $challenge->category) }}">
                    </div>
                    <div class="field">
                        <label for="difficulty">سطح سختی</label>
                        <select id="difficulty" name="difficulty">
                            <option value="">—</option>
                            @foreach ($difficulties as $d)
                                <option value="{{ $d }}" @selected(old('difficulty', $challenge->difficulty) === $d)>{{ ['easy'=>'آسان','medium'=>'متوسط','hard'=>'سخت'][$d] }}</option>
                            @endforeach
                        </select>
                    </div>
                    <div class="field">
                        <label for="sort_order">ترتیب نمایش</label>
                        <input type="number" id="sort_order" name="sort_order" value="{{ old('sort_order', $challenge->sort_order ?? 0) }}">
                    </div>
                    <div class="field switch-row" style="align-self:end">
                        <input type="checkbox" id="is_active" name="is_active" value="1" @checked(old('is_active', $challenge->is_active))>
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
