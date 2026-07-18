<!DOCTYPE html>
<html lang="fa" dir="rtl">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="csrf-token" content="{{ csrf_token() }}">
    <title>@yield('title', 'پنل مدیریت') — ریتمه</title>
    <link rel="stylesheet" href="{{ route('admin.style') }}">
</head>
<body>
@php $admin = auth('admin')->user(); @endphp
<div class="layout">
    <aside class="sidebar" id="sidebar">
        <div class="brand"><span class="dot"></span> ریتمه</div>

        <div class="nav-group-title">عمومی</div>
        <a class="nav-link {{ request()->routeIs('admin.dashboard') ? 'active' : '' }}" href="{{ route('admin.dashboard') }}">داشبورد</a>
        @if (Route::has('admin.users.index'))
            <a class="nav-link {{ request()->routeIs('admin.users.*') ? 'active' : '' }}" href="{{ route('admin.users.index') }}">مدیریت کاربران</a>
        @endif

        @if (Route::has('admin.articles.index'))
            <div class="nav-group-title">محتوا</div>
            <a class="nav-link {{ request()->routeIs('admin.articles.*') ? 'active' : '' }}" href="{{ route('admin.articles.index') }}">مقالات</a>
            <a class="nav-link {{ request()->routeIs('admin.affirmations.*') ? 'active' : '' }}" href="{{ route('admin.affirmations.index') }}">تأکیدات مثبت</a>
            <a class="nav-link {{ request()->routeIs('admin.challenges.*') ? 'active' : '' }}" href="{{ route('admin.challenges.index') }}">چالش‌ها</a>
            <a class="nav-link {{ request()->routeIs('admin.task-templates.*') ? 'active' : '' }}" href="{{ route('admin.task-templates.index') }}">کارهای روزانه</a>
            <a class="nav-link {{ request()->routeIs('admin.pregnancy-weeks.*') ? 'active' : '' }}" href="{{ route('admin.pregnancy-weeks.index') }}">محتوای هفتگی بارداری</a>
            <a class="nav-link {{ request()->routeIs('admin.phase-contents.*') ? 'active' : '' }}" href="{{ route('admin.phase-contents.index') }}">محتوای فازهای چرخه</a>
            <a class="nav-link {{ request()->routeIs('admin.banners.*') ? 'active' : '' }}" href="{{ route('admin.banners.index') }}">بنرها و تبلیغات</a>
        @endif

        @if (Route::has('admin.messages.index'))
            <div class="nav-group-title">پیام‌های هوشمند</div>
            <a class="nav-link {{ request()->routeIs('admin.messages.*') ? 'active' : '' }}" href="{{ route('admin.messages.index') }}">
                پیام‌ها
                @if (($pendingMessages = \App\Models\MessageContent::where('is_approved', false)->count()) > 0)
                    <span class="badge amber">{{ $pendingMessages }}</span>
                @endif
            </a>
        @endif

        <div class="nav-group-title">حساب</div>
        <a class="nav-link {{ request()->routeIs('admin.password.*') ? 'active' : '' }}" href="{{ route('admin.password.edit') }}">تغییر رمز عبور</a>
        @if ($admin?->isSuper() && Route::has('admin.admins.index'))
            <a class="nav-link {{ request()->routeIs('admin.admins.*') ? 'active' : '' }}" href="{{ route('admin.admins.index') }}">مدیریت ادمین‌ها</a>
        @endif
        <form method="POST" action="{{ route('admin.logout') }}" style="margin-top:8px">
            @csrf
            <button type="submit" class="nav-link" style="width:100%;border:0;background:none;cursor:pointer;text-align:start;color:var(--danger)">خروج</button>
        </form>
    </aside>

    <div class="main">
        <div class="topbar">
            <button class="btn ghost menu-toggle" onclick="document.getElementById('sidebar').classList.toggle('open')">☰</button>
            <h1>@yield('title', 'پنل مدیریت')</h1>
            <div class="spacer"></div>
            <div class="who">{{ $admin?->name }} @if($admin?->isSuper())<span class="badge pink">ارشد</span>@endif</div>
        </div>

        <div class="content">
            @include('admin.partials.flash')
            @yield('content')
        </div>
    </div>
</div>
</body>
</html>
