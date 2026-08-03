@extends('admin.layouts.app')

@section('title', 'چالش‌ها')

@section('content')
    <div class="page-actions">
        <a class="btn" href="{{ route('admin.challenge-completions.index') }}">گزارش انجام‌ها</a>
        <div class="spacer"></div>
        <a class="btn primary" href="{{ route('admin.challenges.create') }}">+ چالش جدید</a>
    </div>

    <div class="card">
        <div class="card-head">
            <h2>چالش‌ها</h2>
            <div class="spacer"></div>
            {{-- «روز چرخه» بر اساس همان چیزی فیلتر می‌کند که کاربرِ آن روز می‌بیند: چالش‌های آن بازه به‌علاوه چالش‌های بدون بازه --}}
            <form method="GET" class="btn-row" style="gap:8px; flex-wrap:wrap">
                <input type="text" name="q" value="{{ $filters['q'] }}" placeholder="جستجو در عنوان، توضیح یا دسته" style="width:220px">
                <input type="number" name="cycle_day" min="1" max="{{ $maxCycleDay }}" value="{{ $filters['cycle_day'] }}" placeholder="روز چرخه" style="width:110px">
                <select name="status">
                    <option value="">همه وضعیت‌ها</option>
                    <option value="active" @selected($filters['status'] === 'active')>فعال</option>
                    <option value="inactive" @selected($filters['status'] === 'inactive')>غیرفعال</option>
                </select>
                <button class="btn" type="submit">اعمال</button>
                <a class="btn sm" href="{{ route('admin.challenges.index') }}">حذف فیلتر</a>
            </form>
        </div>
        <div class="card-body tight">
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr><th>#</th><th>عنوان</th><th>روز چرخه</th><th>دسته</th><th>ترتیب</th><th>وضعیت</th><th></th></tr>
                    </thead>
                    <tbody>
                        @forelse ($challenges as $item)
                            <tr>
                                <td>{{ $item->id }}</td>
                                <td class="wrap">{{ $item->title['fa'] ?? '—' }}</td>
                                <td>
                                    <span class="badge {{ $item->isDayTargeted() ? 'pink' : '' }}">{{ $item->cycleDayLabel() }}</span>
                                </td>
                                <td>{{ $item->category ?: '—' }}</td>
                                <td>{{ $item->sort_order }}</td>
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
                            <tr><td colspan="7" class="empty">موردی ثبت نشده است</td></tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    {{ $challenges->links('admin.partials.pagination') }}
@endsection
