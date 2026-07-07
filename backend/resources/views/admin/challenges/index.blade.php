@extends('admin.layouts.app')

@section('title', 'چالش‌ها')

@section('content')
    <div class="page-actions">
        <div class="spacer"></div>
        <a class="btn primary" href="{{ route('admin.challenges.create') }}">+ چالش جدید</a>
    </div>

    <div class="card">
        <div class="card-body tight">
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr><th>#</th><th>عنوان</th><th>فاز</th><th>سختی</th><th>وضعیت</th><th></th></tr>
                    </thead>
                    <tbody>
                        @forelse ($challenges as $item)
                            <tr>
                                <td>{{ $item->id }}</td>
                                <td class="wrap">{{ $item->title['fa'] ?? '—' }}</td>
                                <td>{{ $item->cycle_phase ?: '—' }}</td>
                                <td>{{ $item->difficulty ?: '—' }}</td>
                                <td>@if ($item->is_active)<span class="badge green">فعال</span>@else<span class="badge">غیرفعال</span>@endif</td>
                                <td>
                                    <div class="btn-row">
                                        <a class="btn sm" href="{{ route('admin.challenges.edit', $item) }}">ویرایش</a>
                                        <form method="POST" action="{{ route('admin.challenges.toggle', $item) }}">@csrf<button class="btn sm" type="submit">{{ $item->is_active ? 'غیرفعال' : 'فعال' }}</button></form>
                                        <form method="POST" action="{{ route('admin.challenges.destroy', $item) }}" onsubmit="return confirm('حذف شود؟')">@csrf @method('DELETE')<button class="btn sm danger" type="submit">حذف</button></form>
                                    </div>
                                </td>
                            </tr>
                        @empty
                            <tr><td colspan="6" class="empty">موردی ثبت نشده است</td></tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    {{ $challenges->links('admin.partials.pagination') }}
@endsection
