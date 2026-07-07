@extends('admin.layouts.app')

@php $isEdit = $week->exists; @endphp
@section('title', $isEdit ? 'ویرایش هفته ' . $week->week_number : 'هفته جدید')

@section('content')
    <div class="page-actions">
        <a class="btn ghost" href="{{ route('admin.pregnancy-weeks.index') }}">→ بازگشت</a>
        <div class="spacer"></div>
        @if ($isEdit)
            <form method="POST" action="{{ route('admin.pregnancy-weeks.destroy', $week) }}" onsubmit="return confirm('حذف محتوای این هفته؟')">
                @csrf @method('DELETE')
                <button class="btn danger" type="submit">حذف هفته</button>
            </form>
        @endif
    </div>

    <div class="card">
        <div class="card-body">
            <form method="POST" action="{{ $isEdit ? route('admin.pregnancy-weeks.update', $week) : route('admin.pregnancy-weeks.store') }}">
                @csrf
                @if ($isEdit) @method('PUT') @endif

                <div class="form-grid">
                    <div class="field">
                        <label for="week_number">شماره هفته</label>
                        <input type="number" id="week_number" name="week_number" min="1" max="42" value="{{ old('week_number', $week->week_number) }}" required @if($isEdit) readonly @endif>
                        @error('week_number') <span class="err">{{ $message }}</span> @enderror
                    </div>
                </div>

                @foreach ($fields as $field => $label)
                    <hr style="border:0;border-top:1px solid var(--border);margin:18px 0 6px">
                    <div class="form-grid">
                        <x-admin.bilingual :name="$field" :label="$label" :value="$week->{$field}" type="textarea" />
                    </div>
                @endforeach

                <div class="form-actions">
                    <button class="btn primary" type="submit">{{ $isEdit ? 'ذخیره تغییرات' : 'ایجاد' }}</button>
                </div>
            </form>
        </div>
    </div>
@endsection
