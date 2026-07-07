@extends('admin.layouts.app')

@section('title', 'مدیریت کاربران')

@section('content')
    <div class="card">
        <div class="card-head">
            <h2>کاربران</h2>
            <div class="spacer"></div>
            <form method="GET" class="btn-row" style="gap:8px">
                <input type="text" name="q" value="{{ $search }}" placeholder="جستجو: نام، موبایل، ایمیل" style="width:240px">
                <select name="status" onchange="this.form.submit()">
                    <option value="all" @selected($status===null || $status==='all')>همه</option>
                    <option value="active" @selected($status==='active')>فعال</option>
                    <option value="blocked" @selected($status==='blocked')>مسدود</option>
                </select>
                <button class="btn" type="submit">اعمال</button>
            </form>
        </div>
        <div class="card-body tight">
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr>
                            <th>#</th>
                            <th>نام</th>
                            <th>موبایل</th>
                            <th>اشتراک</th>
                            <th>هدف</th>
                            <th>وضعیت</th>
                            <th>ثبت‌نام</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        @forelse ($users as $user)
                            <tr>
                                <td>{{ $user->id }}</td>
                                <td>{{ $user->name ?: '—' }}</td>
                                <td>{{ $user->mobile ?: '—' }}</td>
                                <td>
                                    @if (($user->profile?->subscription_type ?? 'free') === 'premium')
                                        <span class="badge pink">پرمیوم</span>
                                    @else
                                        <span class="badge">رایگان</span>
                                    @endif
                                </td>
                                <td>{{ ($user->profile?->user_goal ?? 'non_ttc') === 'ttc' ? 'اقدام به بارداری' : 'بدون قصد' }}</td>
                                <td>
                                    @if ($user->blocked_at)
                                        <span class="badge red">مسدود</span>
                                    @else
                                        <span class="badge green">فعال</span>
                                    @endif
                                </td>
                                <td>{{ $user->created_at?->format('Y-m-d') }}</td>
                                <td><a class="btn sm" href="{{ route('admin.users.show', $user) }}">مشاهده</a></td>
                            </tr>
                        @empty
                            <tr><td colspan="8" class="empty">کاربری یافت نشد</td></tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    {{ $users->links('admin.partials.pagination') }}
@endsection
