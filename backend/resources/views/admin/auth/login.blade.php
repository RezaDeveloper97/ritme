@extends('admin.layouts.guest')

@section('title', 'ورود')

@section('content')
    <div class="card">
        <div class="card-body">
            <div class="brand"><span class="dot"></span> ریتمه</div>
            <p class="login-sub">ورود به پنل مدیریت</p>

            @if ($errors->any())
                <div class="alert error">{{ $errors->first() }}</div>
            @endif

            <form method="POST" action="{{ route('admin.login.attempt') }}">
                @csrf
                <div class="field">
                    <label for="email">ایمیل</label>
                    <input type="email" id="email" name="email" value="{{ old('email') }}" required autofocus>
                </div>
                <div class="field">
                    <label for="password">رمز عبور</label>
                    <input type="password" id="password" name="password" required>
                </div>
                <div class="field switch-row">
                    <input type="checkbox" id="remember" name="remember" value="1">
                    <label for="remember" style="margin:0">مرا به خاطر بسپار</label>
                </div>
                <button type="submit" class="btn primary" style="width:100%;margin-top:6px">ورود</button>
            </form>
        </div>
    </div>
    <p class="login-foot">ریتمه — همراه چرخه‌ی تو</p>
@endsection
