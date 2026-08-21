@extends('admin.layouts.app')

@section('title', 'داشبورد')

@section('content')
    @php
        $admin = auth('admin')->user();
        $hour = (int) now()->format('G');
        $greeting = $hour < 5 ? 'شب بخیر' : ($hour < 12 ? 'صبح بخیر' : ($hour < 17 ? 'ظهر بخیر' : 'عصر بخیر'));
    @endphp

    <section class="hero">
        <h2>{{ $greeting }}، {{ $admin?->name }} 🌸</h2>
        <p>یک نگاه کوتاه به وضعیت امروز ریتمه؛ از اینجا می‌توانی کاربران، محتوا و پیام‌های هوشمند را مدیریت کنی.</p>
        <div class="hero-meta">
            <span class="chip">{{ number_format($stats['users_new_today'] ?? 0) }} کاربر جدید امروز</span>
            <span class="chip">{{ number_format($stats['users_new_week'] ?? 0) }} کاربر در ۷ روز اخیر</span>
            @if (($stats['messages_pending'] ?? 0) > 0)
                <span class="chip">{{ number_format($stats['messages_pending']) }} پیام در انتظار تأیید</span>
            @endif
        </div>
    </section>

    <div class="stat-grid">
        <div class="stat">
            <div class="label">@include('admin.partials.icon', ['name' => 'users']) کاربران</div>
            <div class="value">{{ number_format($stats['users']) }}</div>
            <div class="sub">{{ number_format($stats['users_blocked']) }} مسدود</div>
        </div>
        <div class="stat">
            <div class="label">@include('admin.partials.icon', ['name' => 'article']) مقالات</div>
            <div class="value">{{ number_format($stats['articles']) }}</div>
            <div class="sub">محتوای آموزشی منتشرشده</div>
        </div>
        <div class="stat">
            <div class="label">@include('admin.partials.icon', ['name' => 'flag']) چالش‌ها</div>
            <div class="value">{{ number_format($stats['challenges']) }}</div>
            <div class="sub">چالش‌های تعریف‌شده</div>
        </div>
        <div class="stat">
            <div class="label">@include('admin.partials.icon', ['name' => 'sparkle']) تأکیدات مثبت</div>
            <div class="value">{{ number_format($stats['affirmations']) }}</div>
            <div class="sub">جملات انگیزشی روزانه</div>
        </div>
        <div class="stat">
            <div class="label">@include('admin.partials.icon', ['name' => 'task']) کارهای روزانه</div>
            <div class="value">{{ number_format($stats['task_templates']) }}</div>
            <div class="sub">قالب‌های فعال</div>
        </div>
        <div class="stat {{ $stats['messages_pending'] > 0 ? 'is-alert' : '' }}">
            <div class="label">@include('admin.partials.icon', ['name' => 'message']) پیام‌های هوشمند</div>
            <div class="value">{{ number_format($stats['messages']) }}</div>
            <div class="sub">{{ number_format($stats['messages_pending']) }} در انتظار تأیید</div>
        </div>
    </div>

    <div class="card">
        <div class="card-head">
            <h2>کاربران اخیر</h2>
            <div class="spacer"></div>
            @if (Route::has('admin.users.index'))
                <a class="btn sm" href="{{ route('admin.users.index') }}">همه کاربران</a>
            @endif
        </div>
        <div class="card-body tight">
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr>
                            <th>#</th>
                            <th>نام</th>
                            <th>موبایل</th>
                            <th>وضعیت</th>
                            <th>تاریخ ثبت‌نام</th>
                        </tr>
                    </thead>
                    <tbody>
                        @forelse ($recentUsers as $user)
                            <tr>
                                <td>{{ $user->id }}</td>
                                <td>{{ $user->name ?: '—' }}</td>
                                <td>{{ $user->mobile ?: '—' }}</td>
                                <td>
                                    @if ($user->blocked_at)
                                        <span class="badge red">مسدود</span>
                                    @else
                                        <span class="badge green">فعال</span>
                                    @endif
                                </td>
                                <td>{{ $user->created_at?->format('Y-m-d H:i') }}</td>
                            </tr>
                        @empty
                            <tr><td colspan="5" class="empty">کاربری یافت نشد</td></tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>
    </div>
@endsection
