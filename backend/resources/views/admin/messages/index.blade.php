@extends('admin.layouts.app')

@section('title', 'پیام‌های هوشمند')

@section('content')
    <div class="card">
        <div class="card-head">
            <h2>پیام‌های هوشمند</h2>
            <div class="spacer"></div>
            <form method="GET" class="btn-row" style="gap:8px">
                <select name="group" onchange="this.form.submit()">
                    <option value="">همه گروه‌ها</option>
                    @foreach ($groups as $g)
                        <option value="{{ $g }}" @selected($group===$g)>{{ $labels[$g] ?? $g }}</option>
                    @endforeach
                </select>
                <select name="locale" onchange="this.form.submit()">
                    <option value="">هر زبان</option>
                    <option value="fa" @selected($locale==='fa')>فارسی</option>
                    <option value="en" @selected($locale==='en')>English</option>
                </select>
                <select name="status" onchange="this.form.submit()">
                    <option value="">هر وضعیت</option>
                    <option value="approved" @selected($status==='approved')>تأییدشده</option>
                    <option value="pending" @selected($status==='pending')>در انتظار تأیید</option>
                </select>
            </form>
        </div>
        <div class="card-body tight">
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr><th>گروه</th><th>کلید</th><th>زبان</th><th>پیش‌نمایش</th><th>وضعیت</th><th></th></tr>
                    </thead>
                    <tbody>
                        @forelse ($items as $item)
                            <tr>
                                <td>{{ $labels[$item->group] ?? $item->group }}</td>
                                <td dir="ltr">{{ $item->item_key }}</td>
                                <td>{{ $item->locale }}</td>
                                @php $vals = array_values($item->payload ?? []); $first = $vals[0] ?? ''; @endphp
                                <td class="wrap muted" style="max-width:360px">{{ \Illuminate\Support\Str::limit(is_array($first) ? implode('، ', $first) : (string) $first, 80) }}</td>
                                <td>
                                    @if ($item->is_approved)<span class="badge green">تأیید</span>@else<span class="badge amber">در انتظار</span>@endif
                                    @unless ($item->is_active)<span class="badge red">غیرفعال</span>@endunless
                                </td>
                                <td>
                                    <div class="btn-row">
                                        <a class="btn sm" href="{{ route('admin.messages.edit', $item) }}">ویرایش</a>
                                        <form method="POST" action="{{ route('admin.messages.approve', $item) }}">@csrf<button class="btn sm" type="submit">{{ $item->is_approved ? 'لغو تأیید' : 'تأیید' }}</button></form>
                                        <form method="POST" action="{{ route('admin.messages.toggle', $item) }}">@csrf<button class="btn sm" type="submit">{{ $item->is_active ? 'غیرفعال' : 'فعال' }}</button></form>
                                    </div>
                                </td>
                            </tr>
                        @empty
                            <tr><td colspan="6" class="empty">پیامی یافت نشد. برای پر شدن، seeder را اجرا کنید.</td></tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    {{ $items->links('admin.partials.pagination') }}
@endsection
