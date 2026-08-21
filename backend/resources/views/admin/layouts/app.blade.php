<!DOCTYPE html>
<html lang="fa" dir="rtl">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="csrf-token" content="{{ csrf_token() }}">
    <meta name="theme-color" content="#e11d67">
    <title>@yield('title', 'پنل مدیریت') — ریتمه</title>
    <link rel="stylesheet" href="{{ route('admin.style') }}">
</head>
<body>
@php $admin = auth('admin')->user(); @endphp
<div class="layout">
    <aside class="sidebar" id="sidebar">
        <div class="brand">
            <span class="dot"></span>
            <span>ریتمه<small>پنل مدیریت</small></span>
        </div>

        <div class="nav-group-title">عمومی</div>
        <a class="nav-link {{ request()->routeIs('admin.dashboard') ? 'active' : '' }}" href="{{ route('admin.dashboard') }}">
            @include('admin.partials.icon', ['name' => 'dashboard']) داشبورد
        </a>
        @if (Route::has('admin.users.index'))
            <a class="nav-link {{ request()->routeIs('admin.users.*') ? 'active' : '' }}" href="{{ route('admin.users.index') }}">
                @include('admin.partials.icon', ['name' => 'users']) مدیریت کاربران
            </a>
        @endif

        @if (Route::has('admin.articles.index'))
            <div class="nav-group-title">محتوا</div>
            <a class="nav-link {{ request()->routeIs('admin.articles.*') ? 'active' : '' }}" href="{{ route('admin.articles.index') }}">
                @include('admin.partials.icon', ['name' => 'article']) مقالات
            </a>
            <a class="nav-link {{ request()->routeIs('admin.affirmations.*') ? 'active' : '' }}" href="{{ route('admin.affirmations.index') }}">
                @include('admin.partials.icon', ['name' => 'sparkle']) تأکیدات مثبت
            </a>
            <a class="nav-link {{ request()->routeIs('admin.challenges.*') ? 'active' : '' }}" href="{{ route('admin.challenges.index') }}">
                @include('admin.partials.icon', ['name' => 'flag']) چالش‌ها
            </a>
            <a class="nav-link {{ request()->routeIs('admin.challenge-completions.*') ? 'active' : '' }}" href="{{ route('admin.challenge-completions.index') }}">
                @include('admin.partials.icon', ['name' => 'check']) انجام چالش‌ها
            </a>
            <a class="nav-link {{ request()->routeIs('admin.task-templates.*') ? 'active' : '' }}" href="{{ route('admin.task-templates.index') }}">
                @include('admin.partials.icon', ['name' => 'task']) کارهای روزانه
            </a>
            <a class="nav-link {{ request()->routeIs('admin.pregnancy-weeks.*') ? 'active' : '' }}" href="{{ route('admin.pregnancy-weeks.index') }}">
                @include('admin.partials.icon', ['name' => 'baby']) محتوای هفتگی بارداری
            </a>
            <a class="nav-link {{ request()->routeIs('admin.phase-contents.*') ? 'active' : '' }}" href="{{ route('admin.phase-contents.index') }}">
                @include('admin.partials.icon', ['name' => 'moon']) محتوای فازهای چرخه
            </a>
            <a class="nav-link {{ request()->routeIs('admin.recommendations.*') ? 'active' : '' }}" href="{{ route('admin.recommendations.index') }}">
                @include('admin.partials.icon', ['name' => 'idea']) توصیه‌های امروز
            </a>
            <a class="nav-link {{ request()->routeIs('admin.banners.*') ? 'active' : '' }}" href="{{ route('admin.banners.index') }}">
                @include('admin.partials.icon', ['name' => 'banner']) بنرها و تبلیغات
            </a>
            <a class="nav-link {{ request()->routeIs('admin.info-sections.*') ? 'active' : '' }}" href="{{ route('admin.info-sections.index') }}">
                @include('admin.partials.icon', ['name' => 'life-ring']) راهنما و صفحات متنی
            </a>
        @endif

        @if (Route::has('admin.messages.index'))
            <div class="nav-group-title">پیام‌های هوشمند</div>
            <a class="nav-link {{ request()->routeIs('admin.messages.*') ? 'active' : '' }}" href="{{ route('admin.messages.index') }}">
                @include('admin.partials.icon', ['name' => 'message']) پیام‌ها
                @if (($pendingMessages = \App\Models\MessageContent::where('is_approved', false)->count()) > 0)
                    <span class="badge amber">{{ $pendingMessages }}</span>
                @endif
            </a>
        @endif

        @if ($admin?->isSuper() && Route::has('admin.languages.index'))
            <div class="nav-group-title">تنظیمات</div>
            <a class="nav-link {{ request()->routeIs('admin.languages.*') ? 'active' : '' }}" href="{{ route('admin.languages.index') }}">
                @include('admin.partials.icon', ['name' => 'language']) زبان‌ها
            </a>
        @endif

        <div class="nav-group-title">حساب</div>
        <a class="nav-link {{ request()->routeIs('admin.password.*') ? 'active' : '' }}" href="{{ route('admin.password.edit') }}">
            @include('admin.partials.icon', ['name' => 'key']) تغییر رمز عبور
        </a>
        @if ($admin?->isSuper() && Route::has('admin.admins.index'))
            <a class="nav-link {{ request()->routeIs('admin.admins.*') ? 'active' : '' }}" href="{{ route('admin.admins.index') }}">
                @include('admin.partials.icon', ['name' => 'shield']) مدیریت ادمین‌ها
            </a>
        @endif

        <div class="sidebar-sep"></div>
        <form method="POST" action="{{ route('admin.logout') }}" style="margin-top:8px">
            @csrf
            <button type="submit" class="nav-link danger" style="width:100%;border:0;background:none;cursor:pointer;text-align:start;font-family:inherit;font-size:13.5px">
                @include('admin.partials.icon', ['name' => 'logout']) خروج
            </button>
        </form>
    </aside>

    <div class="scrim" id="scrim" onclick="toggleNav(false)"></div>

    <div class="main">
        <div class="topbar">
            <button class="btn ghost menu-toggle" onclick="toggleNav()" aria-label="منو">
                @include('admin.partials.icon', ['name' => 'menu'])
            </button>
            <h1>@yield('title', 'پنل مدیریت')</h1>
            <div class="spacer"></div>
            <div class="who">
                <span class="avatar">{{ mb_substr($admin?->name ?: '؟', 0, 1) }}</span>
                <span class="name">{{ $admin?->name }}</span>
                @if($admin?->isSuper())<span class="badge pink">ارشد</span>@endif
            </div>
        </div>

        <div class="content">
            @include('admin.partials.flash')
            @yield('content')
        </div>
    </div>
</div>
<script>
    function toggleNav(force) {
        var sidebar = document.getElementById('sidebar');
        var open = force === undefined ? !sidebar.classList.contains('open') : force;
        sidebar.classList.toggle('open', open);
        document.body.classList.toggle('nav-open', open);
    }
    // Escape closes the mobile drawer — never trap the user inside the menu.
    document.addEventListener('keydown', function (e) { if (e.key === 'Escape') toggleNav(false); });
</script>
</body>
</html>
