@extends('admin.layouts.app')

@section('title', 'کارهای روزانه')

@section('content')
    <div class="page-actions">
        <div class="spacer"></div>
        <a class="btn primary" href="{{ route('admin.task-templates.create') }}">+ کار جدید</a>
    </div>

    <div class="card">
        <div class="card-body tight">
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr><th>#</th><th>شناسه</th><th>عنوان</th><th>دسته</th><th>فاز</th><th>وضعیت</th><th></th></tr>
                    </thead>
                    <tbody>
                        @forelse ($tasks as $item)
                            <tr>
                                <td>{{ $item->id }}</td>
                                <td dir="ltr">{{ $item->key }}</td>
                                <td class="wrap">{{ $item->title['fa'] ?? '—' }}</td>
                                <td>{{ $item->category }}</td>
                                <td>{{ $item->cycle_phase ?: '—' }}</td>
                                <td>@if ($item->is_active)<span class="badge green">فعال</span>@else<span class="badge">غیرفعال</span>@endif</td>
                                <td>
                                    <div class="btn-row">
                                        <a class="btn sm" href="{{ route('admin.task-templates.edit', $item) }}">ویرایش</a>
                                        <form method="POST" action="{{ route('admin.task-templates.toggle', $item) }}">@csrf<button class="btn sm" type="submit">{{ $item->is_active ? 'غیرفعال' : 'فعال' }}</button></form>
                                        <form method="POST" action="{{ route('admin.task-templates.destroy', $item) }}" onsubmit="return confirm('حذف شود؟')">@csrf @method('DELETE')<button class="btn sm danger" type="submit">حذف</button></form>
                                    </div>
                                </td>
                            </tr>
                        @empty
                            <tr><td colspan="7" class="empty">موردی ثبت نشده است</td></tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    {{ $tasks->links('admin.partials.pagination') }}
@endsection
