@extends('admin.layouts.app')

@section('title', 'تغییر رمز عبور')

@section('content')
    <div class="card" style="max-width:520px">
        <div class="card-head"><h2>تغییر رمز عبور</h2></div>
        <div class="card-body">
            <form method="POST" action="{{ route('admin.password.update') }}">
                @csrf
                @method('PUT')
                <div class="field">
                    <label for="current_password">رمز عبور فعلی</label>
                    <input type="password" id="current_password" name="current_password" required>
                    @error('current_password') <span class="err">{{ $message }}</span> @enderror
                </div>
                <div class="field">
                    <label for="password">رمز عبور جدید</label>
                    <input type="password" id="password" name="password" required>
                    <span class="hint">حداقل ۸ کاراکتر</span>
                    @error('password') <span class="err">{{ $message }}</span> @enderror
                </div>
                <div class="field">
                    <label for="password_confirmation">تکرار رمز عبور جدید</label>
                    <input type="password" id="password_confirmation" name="password_confirmation" required>
                </div>
                <div class="form-actions">
                    <button type="submit" class="btn primary">ذخیره</button>
                </div>
            </form>
        </div>
    </div>
@endsection
