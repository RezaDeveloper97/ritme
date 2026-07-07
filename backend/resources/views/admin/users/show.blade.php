@extends('admin.layouts.app')

@section('title', 'کاربر: ' . ($user->name ?: $user->mobile))

@section('content')
    <div class="page-actions">
        <a class="btn ghost" href="{{ route('admin.users.index') }}">→ بازگشت به لیست</a>
        <div class="spacer"></div>
        @if ($user->blocked_at)
            <form method="POST" action="{{ route('admin.users.unblock', $user) }}">
                @csrf
                <button class="btn" type="submit">رفع مسدودیت</button>
            </form>
        @else
            <form method="POST" action="{{ route('admin.users.block', $user) }}" onsubmit="return confirm('این کاربر مسدود شود؟')">
                @csrf
                <button class="btn danger" type="submit">مسدودسازی</button>
            </form>
        @endif
        <form method="POST" action="{{ route('admin.users.destroy', $user) }}" onsubmit="return confirm('حذف دائمی این کاربر و همه داده‌هایش؟ این عمل قابل بازگشت نیست.')">
            @csrf
            @method('DELETE')
            <button class="btn danger" type="submit">حذف کاربر</button>
        </form>
    </div>

    <div class="stat-grid">
        <div class="stat"><div class="label">لاگ سلامت</div><div class="value">{{ $stats['health_logs'] }}</div></div>
        <div class="stat"><div class="label">یادآورها</div><div class="value">{{ $stats['reminders'] }}</div></div>
        <div class="stat"><div class="label">اعلان‌ها</div><div class="value">{{ $stats['notifications'] }}</div></div>
        <div class="stat"><div class="label">محاسبات سیکل</div><div class="value">{{ $stats['cycle_calculations'] }}</div></div>
    </div>

    <div class="card">
        <div class="card-head">
            <h2>اطلاعات کاربر</h2>
            <div class="spacer"></div>
            @if ($user->blocked_at)
                <span class="badge red">مسدود شده در {{ $user->blocked_at->format('Y-m-d') }}</span>
            @else
                <span class="badge green">فعال</span>
            @endif
        </div>
        <div class="card-body">
            <form method="POST" action="{{ route('admin.users.update', $user) }}">
                @csrf
                @method('PUT')
                <div class="form-grid">
                    <div class="field">
                        <label>موبایل</label>
                        <input type="text" value="{{ $user->mobile ?: '—' }}" disabled>
                    </div>
                    <div class="field">
                        <label for="name">نام</label>
                        <input type="text" id="name" name="name" value="{{ old('name', $user->name) }}">
                    </div>
                    <div class="field">
                        <label for="subscription_type">نوع اشتراک</label>
                        <select id="subscription_type" name="subscription_type">
                            @foreach ($goals as $g)
                                <option value="{{ $g->value }}" @selected(old('subscription_type', $user->profile?->subscription_type ?? 'free') === $g->value)>{{ $g->label('fa') }}</option>
                            @endforeach
                        </select>
                    </div>
                    <div class="field">
                        <label for="user_goal">هدف کاربر</label>
                        <select id="user_goal" name="user_goal">
                            @foreach ($userGoals as $g)
                                <option value="{{ $g->value }}" @selected(old('user_goal', $user->profile?->user_goal ?? 'non_ttc') === $g->value)>{{ $g->label('fa') }}</option>
                            @endforeach
                        </select>
                    </div>
                </div>
                <div class="form-actions">
                    <button class="btn primary" type="submit">ذخیره تغییرات</button>
                </div>
            </form>
        </div>
    </div>

    <div class="card">
        <div class="card-head"><h2>پروفایل سلامت</h2></div>
        <div class="card-body">
            @if ($user->profile)
                <div class="form-grid">
                    <div class="field"><label>تاریخ تولد</label><input type="text" value="{{ $user->profile->birthday?->format('Y-m-d') ?: '—' }}" disabled></div>
                    <div class="field"><label>آخرین پریود</label><input type="text" value="{{ $user->profile->last_period_start?->format('Y-m-d') ?: '—' }}" disabled></div>
                    <div class="field"><label>طول سیکل</label><input type="text" value="{{ $user->profile->cycle_duration ?: '—' }}" disabled></div>
                    <div class="field"><label>مدت پریود</label><input type="text" value="{{ $user->profile->period_duration ?: '—' }}" disabled></div>
                </div>
            @else
                <p class="muted">این کاربر هنوز پروفایل کامل نکرده است.</p>
            @endif
        </div>
    </div>
@endsection
