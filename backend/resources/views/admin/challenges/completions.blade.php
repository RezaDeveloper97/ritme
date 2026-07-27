@extends('admin.layouts.app')

@section('title', 'انجام چالش‌ها')

@section('content')
    <div class="page-actions">
        <div class="spacer"></div>
        <a class="btn" href="{{ route('admin.challenges.index') }}">مدیریت چالش‌ها</a>
    </div>

    <div class="card">
        <div class="card-head">
            <h2>انجام چالش‌ها</h2>
            <div class="spacer"></div>
            <form method="GET" class="btn-row" style="gap:8px; flex-wrap:wrap">
                <select name="challenge_id">
                    <option value="">همه چالش‌ها</option>
                    @foreach ($challenges as $item)
                        <option value="{{ $item->id }}" @selected((string) $challengeId === (string) $item->id)>
                            {{ $item->title['fa'] ?? ('#' . $item->id) }}
                        </option>
                    @endforeach
                </select>
                <input type="text" name="q" value="{{ $search }}" placeholder="جستجوی کاربر: نام یا موبایل" style="width:200px">
                <input type="date" name="from" value="{{ $from }}">
                <input type="date" name="to" value="{{ $to }}">
                <button class="btn" type="submit">اعمال</button>
                <a class="btn sm" href="{{ route('admin.challenge-completions.index') }}">حذف فیلتر</a>
            </form>
        </div>
        <div class="card-body">
            <div class="btn-row" style="gap:12px; flex-wrap:wrap">
                <span class="badge green">مجموع انجام‌ها: {{ number_format($stats['total']) }}</span>
                <span class="badge">کاربران یکتا: {{ number_format($stats['users']) }}</span>
                <span class="badge pink">امروز: {{ number_format($stats['today']) }}</span>
            </div>
        </div>
    </div>

    <div class="card">
        <div class="card-head"><h2>به تفکیک چالش</h2></div>
        <div class="card-body tight">
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr><th>چالش</th><th>تعداد انجام</th><th>کاربران یکتا</th></tr>
                    </thead>
                    <tbody>
                        @forelse ($perChallenge as $row)
                            <tr>
                                <td class="wrap">{{ $row->challenge?->title['fa'] ?? ('#' . $row->challenge_id) }}</td>
                                <td>{{ number_format($row->completions) }}</td>
                                <td>{{ number_format($row->users) }}</td>
                            </tr>
                        @empty
                            <tr><td colspan="3" class="empty">موردی ثبت نشده است</td></tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    <div class="card">
        <div class="card-head"><h2>آخرین انجام‌ها</h2></div>
        <div class="card-body tight">
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr><th>#</th><th>کاربر</th><th>موبایل</th><th>چالش</th><th>سختی</th><th>تاریخ</th></tr>
                    </thead>
                    <tbody>
                        @forelse ($completions as $row)
                            <tr>
                                <td>{{ $row->id }}</td>
                                <td>
                                    @if ($row->user)
                                        <a href="{{ route('admin.users.show', $row->user) }}">{{ $row->user->name ?: ('کاربر #' . $row->user->id) }}</a>
                                    @else
                                        —
                                    @endif
                                </td>
                                <td>{{ $row->user?->mobile ?: '—' }}</td>
                                <td class="wrap">{{ $row->challenge?->title['fa'] ?? '—' }}</td>
                                <td>{{ $row->challenge?->difficulty ?: '—' }}</td>
                                <td>{{ $row->completion_date?->format('Y-m-d') }}</td>
                            </tr>
                        @empty
                            <tr><td colspan="6" class="empty">موردی ثبت نشده است</td></tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    {{ $completions->links('admin.partials.pagination') }}
@endsection
