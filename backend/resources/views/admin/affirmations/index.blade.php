@extends('admin.layouts.app')

@section('title', 'تأکیدات مثبت')

@section('content')
    <div class="page-actions">
        <div class="spacer"></div>
        <a class="btn primary" href="{{ route('admin.affirmations.create') }}">+ تأکید جدید</a>
    </div>

    <div class="card">
        <div class="card-body tight">
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr><th>#</th><th>متن</th><th>فاز</th><th>وضعیت</th><th>ترتیب</th><th></th></tr>
                    </thead>
                    <tbody>
                        @forelse ($affirmations as $item)
                            <tr>
                                <td>{{ $item->id }}</td>
                                <td class="wrap">{{ $item->text['fa'] ?? '—' }}</td>
                                <td>{{ $item->cycle_phase ?: '—' }}</td>
                                <td>@if ($item->is_active)<span class="badge green">فعال</span>@else<span class="badge">غیرفعال</span>@endif</td>
                                <td>{{ $item->sort_order }}</td>
                                <td>
                                    <div class="btn-row">
                                        <a class="btn sm" href="{{ route('admin.affirmations.edit', $item) }}">ویرایش</a>
                                        <form method="POST" action="{{ route('admin.affirmations.toggle', $item) }}">@csrf<button class="btn sm" type="submit">{{ $item->is_active ? 'غیرفعال' : 'فعال' }}</button></form>
                                        <form method="POST" action="{{ route('admin.affirmations.destroy', $item) }}" onsubmit="return confirm('حذف شود؟')">@csrf @method('DELETE')<button class="btn sm danger" type="submit">حذف</button></form>
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

    {{ $affirmations->links('admin.partials.pagination') }}
@endsection
