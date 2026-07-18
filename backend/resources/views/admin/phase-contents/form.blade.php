@extends('admin.layouts.app')

@php $isEdit = $content->exists; @endphp
@section('title', $isEdit ? 'ویرایش محتوای فاز' : 'محتوای فاز جدید')

@section('content')
    <div class="page-actions">
        <a class="btn ghost" href="{{ route('admin.phase-contents.index') }}">→ بازگشت</a>
        <div class="spacer"></div>
        @if ($isEdit)
            <form method="POST" action="{{ route('admin.phase-contents.destroy', $content) }}" onsubmit="return confirm('حذف محتوای این فاز؟')">
                @csrf @method('DELETE')
                <button class="btn danger" type="submit">حذف فاز</button>
            </form>
        @endif
    </div>

    <div class="card">
        <div class="card-body">
            <form method="POST" action="{{ $isEdit ? route('admin.phase-contents.update', $content) : route('admin.phase-contents.store') }}">
                @csrf
                @if ($isEdit) @method('PUT') @endif

                <div class="form-grid">
                    <div class="field">
                        <label for="phase">فاز چرخه</label>
                        @if ($isEdit)
                            <input type="text" id="phase" value="{{ \App\Enums\CycleSubphase::from($content->phase)->label('fa') }}" readonly>
                            <input type="hidden" name="phase" value="{{ $content->phase }}">
                        @else
                            <select id="phase" name="phase" required>
                                @foreach ($phases as $p)
                                    <option value="{{ $p->value }}" @selected(old('phase', $content->phase) === $p->value)>{{ $p->label('fa') }}</option>
                                @endforeach
                            </select>
                        @endif
                        @error('phase') <span class="err">{{ $message }}</span> @enderror
                    </div>
                </div>

                @foreach ($fields as $field => $label)
                    <hr style="border:0;border-top:1px solid var(--border);margin:18px 0 6px">
                    <div class="form-grid">
                        <x-admin.bilingual :name="$field" :label="$label" :value="$content->{$field}" type="textarea" />
                    </div>
                @endforeach

                <div class="form-actions">
                    <button class="btn primary" type="submit">{{ $isEdit ? 'ذخیره تغییرات' : 'ایجاد' }}</button>
                </div>
            </form>
        </div>
    </div>
@endsection
