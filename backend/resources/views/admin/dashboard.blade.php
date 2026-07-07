@extends('admin.layouts.app')

@section('title', 'داشبورد')

@section('content')
    <div class="stat-grid">
        <div class="stat">
            <div class="label">کاربران</div>
            <div class="value">{{ number_format($stats['users']) }}</div>
            <div class="sub">{{ number_format($stats['users_blocked']) }} مسدود</div>
        </div>
        <div class="stat">
            <div class="label">مقالات</div>
            <div class="value">{{ number_format($stats['articles']) }}</div>
        </div>
        <div class="stat">
            <div class="label">چالش‌ها</div>
            <div class="value">{{ number_format($stats['challenges']) }}</div>
        </div>
        <div class="stat">
            <div class="label">تأکیدات مثبت</div>
            <div class="value">{{ number_format($stats['affirmations']) }}</div>
        </div>
        <div class="stat">
            <div class="label">کارهای روزانه</div>
            <div class="value">{{ number_format($stats['task_templates']) }}</div>
        </div>
        <div class="stat">
            <div class="label">پیام‌های هوشمند</div>
            <div class="value">{{ number_format($stats['messages']) }}</div>
            <div class="sub">{{ number_format($stats['messages_pending']) }} در انتظار تأیید</div>
        </div>
    </div>

    <div class="card">
        <div class="card-head"><h2>کاربران اخیر</h2></div>
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
