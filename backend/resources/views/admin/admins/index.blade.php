@extends('admin.layouts.app')

@section('title', 'مدیریت ادمین‌ها')

@section('content')
    <div class="page-actions">
        <div class="spacer"></div>
        <a class="btn primary" href="{{ route('admin.admins.create') }}">+ ادمین جدید</a>
    </div>

    <div class="card">
        <div class="card-body tight">
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr><th>#</th><th>نام</th><th>ایمیل</th><th>نقش</th><th>وضعیت</th><th>آخرین ورود</th><th></th></tr>
                    </thead>
                    <tbody>
                        @foreach ($admins as $item)
                            <tr>
                                <td>{{ $item->id }}</td>
                                <td>{{ $item->name }}</td>
                                <td dir="ltr">{{ $item->email }}</td>
                                <td>@if ($item->isSuper())<span class="badge pink">ارشد</span>@else<span class="badge">ویرایشگر</span>@endif</td>
                                <td>@if ($item->is_active)<span class="badge green">فعال</span>@else<span class="badge red">غیرفعال</span>@endif</td>
                                <td>{{ $item->last_login_at?->format('Y-m-d H:i') ?: '—' }}</td>
                                <td>
                                    <div class="btn-row">
                                        <a class="btn sm" href="{{ route('admin.admins.edit', $item) }}">ویرایش</a>
                                        @if ($item->id !== auth('admin')->id())
                                            <form method="POST" action="{{ route('admin.admins.destroy', $item) }}" onsubmit="return confirm('حذف شود؟')">@csrf @method('DELETE')<button class="btn sm danger" type="submit">حذف</button></form>
                                        @endif
                                    </div>
                                </td>
                            </tr>
                        @endforeach
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    {{ $admins->links('admin.partials.pagination') }}
@endsection
