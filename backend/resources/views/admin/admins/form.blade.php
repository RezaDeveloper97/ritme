@extends('admin.layouts.app')

@php $isEdit = $admin->exists; $isSelf = $isEdit && $admin->id === auth('admin')->id(); @endphp
@section('title', $isEdit ? 'ویرایش ادمین' : 'ادمین جدید')

@section('content')
    <div class="page-actions"><a class="btn ghost" href="{{ route('admin.admins.index') }}">→ بازگشت</a></div>

    <div class="card" style="max-width:640px">
        <div class="card-body">
            <form method="POST" action="{{ $isEdit ? route('admin.admins.update', $admin) : route('admin.admins.store') }}">
                @csrf
                @if ($isEdit) @method('PUT') @endif

                <div class="form-grid">
                    <div class="field">
                        <label for="name">نام</label>
                        <input type="text" id="name" name="name" value="{{ old('name', $admin->name) }}" required>
                        @error('name') <span class="err">{{ $message }}</span> @enderror
                    </div>
                    <div class="field">
                        <label for="email">ایمیل</label>
                        <input type="email" id="email" name="email" dir="ltr" value="{{ old('email', $admin->email) }}" required>
                        @error('email') <span class="err">{{ $message }}</span> @enderror
                    </div>
                    <div class="field">
                        <label for="password">رمز عبور @if($isEdit)<span class="hint">(خالی = بدون تغییر)</span>@endif</label>
                        <input type="password" id="password" name="password" @if(!$isEdit) required @endif>
                        @error('password') <span class="err">{{ $message }}</span> @enderror
                    </div>
                    <div class="field">
                        <label for="password_confirmation">تکرار رمز عبور</label>
                        <input type="password" id="password_confirmation" name="password_confirmation">
                    </div>
                    <div class="field">
                        <label for="role">نقش</label>
                        <select id="role" name="role" @if($isSelf) disabled @endif>
                            <option value="editor" @selected(old('role', $admin->role) === 'editor')>ویرایشگر (فقط محتوا)</option>
                            <option value="super" @selected(old('role', $admin->role) === 'super')>ارشد (دسترسی کامل)</option>
                        </select>
                        @if ($isSelf)<span class="hint">نمی‌توانید نقش خودتان را تغییر دهید</span>@endif
                    </div>
                    <div class="field switch-row" style="align-self:end">
                        <input type="checkbox" id="is_active" name="is_active" value="1" @checked(old('is_active', $admin->is_active ?? true)) @if($isSelf) disabled @endif>
                        <label for="is_active" style="margin:0">فعال</label>
                    </div>
                </div>

                <div class="form-actions">
                    <button class="btn primary" type="submit">{{ $isEdit ? 'ذخیره تغییرات' : 'ایجاد ادمین' }}</button>
                </div>
            </form>
        </div>
    </div>
@endsection
