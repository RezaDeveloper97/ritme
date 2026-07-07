@extends('admin.layouts.app')

@php $isEdit = $affirmation->exists; @endphp
@section('title', $isEdit ? 'ویرایش تأکید' : 'تأکید جدید')

@section('content')
    <div class="page-actions"><a class="btn ghost" href="{{ route('admin.affirmations.index') }}">→ بازگشت</a></div>

    <div class="card">
        <div class="card-body">
            <form method="POST" action="{{ $isEdit ? route('admin.affirmations.update', $affirmation) : route('admin.affirmations.store') }}">
                @csrf
                @if ($isEdit) @method('PUT') @endif

                <div class="form-grid">
                    <x-admin.bilingual name="text" label="متن تأکید" :value="$affirmation->text" type="textarea" required />
                    @include('admin.partials.phase-select', ['selected' => $affirmation->cycle_phase, 'phases' => $phases])
                    <div class="field">
                        <label for="sort_order">ترتیب نمایش</label>
                        <input type="number" id="sort_order" name="sort_order" value="{{ old('sort_order', $affirmation->sort_order ?? 0) }}">
                    </div>
                    <div class="field switch-row" style="align-self:end">
                        <input type="checkbox" id="is_active" name="is_active" value="1" @checked(old('is_active', $affirmation->is_active))>
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
