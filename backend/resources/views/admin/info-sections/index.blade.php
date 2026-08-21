@extends('admin.layouts.app')

@section('title', 'صفحات متنی — ' . \App\Models\InfoSection::groupLabel($group))

@section('content')
    <div class="tabs">
        @foreach (\App\Models\InfoSection::GROUPS as $tab)
            <a class="tab {{ $tab === $group ? 'active' : '' }}"
               href="{{ route('admin.info-sections.index', ['group' => $tab]) }}">
                {{ \App\Models\InfoSection::groupLabel($tab) }}
            </a>
        @endforeach
    </div>

    <div class="page-actions">
        <div class="spacer"></div>
        <a class="btn primary" href="{{ route('admin.info-sections.create', ['group' => $group]) }}">+ باکس جدید</a>
    </div>

    <div class="card">
        <div class="card-body tight">
            <p class="hint" style="margin:0 0 12px">
                هر ردیف یک باکس در صفحه «{{ \App\Models\InfoSection::groupLabel($group) }}» اپ است.
                باکس‌های فعال به ترتیب «ترتیب نمایش» به کاربر نشان داده می‌شوند.
                @if ($group === \App\Models\InfoSection::GROUP_HELP)
                    برای باکس پشتیبانی می‌توانید یک دکمه تماس (ایمیل، تلفن یا لینک) هم تعریف کنید.
                @endif
            </p>
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr><th>#</th><th>عنوان</th><th>متن</th><th>دکمه</th><th>وضعیت</th><th>ترتیب</th><th></th></tr>
                    </thead>
                    <tbody>
                        @forelse ($sections as $item)
                            <tr>
                                <td>{{ $item->id }}</td>
                                <td class="wrap">{{ $item->localized('heading') ?? '—' }}</td>
                                <td class="wrap">{{ \Illuminate\Support\Str::limit($item->localized('body') ?? '', 90) }}</td>
                                <td class="wrap">
                                    @if ($item->link_url)
                                        <span class="badge teal">{{ $item->localized('link_label') ?? 'لینک' }}</span>
                                        <div class="hint" dir="ltr" style="margin-top:4px">{{ \Illuminate\Support\Str::limit($item->link_url, 40) }}</div>
                                    @else
                                        —
                                    @endif
                                </td>
                                <td>@if ($item->is_active)<span class="badge green">فعال</span>@else<span class="badge">غیرفعال</span>@endif</td>
                                <td>{{ $item->sort_order }}</td>
                                <td>
                                    <div class="btn-row">
                                        <a class="btn sm" href="{{ route('admin.info-sections.edit', $item) }}">ویرایش</a>
                                        <form method="POST" action="{{ route('admin.info-sections.toggle', $item) }}">@csrf<button class="btn sm" type="submit">{{ $item->is_active ? 'غیرفعال' : 'فعال' }}</button></form>
                                        <form method="POST" action="{{ route('admin.info-sections.destroy', $item) }}" onsubmit="return confirm('حذف شود؟')">@csrf @method('DELETE')<button class="btn sm danger" type="submit">حذف</button></form>
                                    </div>
                                </td>
                            </tr>
                        @empty
                            <tr><td colspan="7" class="empty">برای این صفحه هنوز باکسی ثبت نشده است</td></tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    {{ $sections->links('admin.partials.pagination') }}
@endsection
