@extends('admin.layouts.app')

@section('title', 'زبان‌ها')

@section('content')
    <div class="page-actions">
        <div class="spacer"></div>
        <a class="btn primary" href="{{ route('admin.languages.create') }}">+ زبان جدید</a>
    </div>

    <div class="card">
        <div class="card-body tight">
            <p class="hint" style="margin:0 0 12px">
                هر زبانی که اینجا اضافه شود، به‌صورت خودکار در تمام فرم‌های محتوای پنل (مقاله، چالش، بنر، صفحات متنی و …)
                یک فیلد جدید می‌گیرد، فایل‌های ترجمه‌اش ساخته می‌شود و در انتخاب زبانِ اپ نمایش داده می‌شود.
                فقط زبان پیش‌فرض اجباری است؛ هر رشته‌ای که در زبان دیگری ترجمه نشده باشد، به زبان پیش‌فرض برمی‌گردد.
            </p>
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr><th>#</th><th>کد</th><th>نام</th><th>نام انگلیسی</th><th>جهت</th><th>وضعیت</th><th>ترتیب</th><th></th></tr>
                    </thead>
                    <tbody>
                        @forelse ($languages as $language)
                            <tr>
                                <td>{{ $language->id }}</td>
                                <td dir="ltr">{{ $language->code }}</td>
                                <td class="wrap">
                                    {{ $language->name }}
                                    @if ($language->is_default)<span class="badge pink">پیش‌فرض</span>@endif
                                </td>
                                <td class="wrap" dir="ltr">{{ $language->english_name }}</td>
                                <td>{{ $language->direction->label('fa') }}</td>
                                <td>@if ($language->is_active)<span class="badge green">فعال</span>@else<span class="badge">غیرفعال</span>@endif</td>
                                <td>{{ $language->sort_order }}</td>
                                <td>
                                    <div class="btn-row">
                                        <a class="btn sm" href="{{ route('admin.languages.translations.index', $language) }}">ترجمه‌ها</a>
                                        <a class="btn sm" href="{{ route('admin.languages.edit', $language) }}">ویرایش</a>
                                        <form method="POST" action="{{ route('admin.languages.toggle', $language) }}">@csrf<button class="btn sm" type="submit">{{ $language->is_active ? 'غیرفعال' : 'فعال' }}</button></form>
                                        @unless ($language->is_default)
                                            <form method="POST" action="{{ route('admin.languages.destroy', $language) }}" onsubmit="return confirm('زبان و همه‌ی فایل‌های ترجمه‌اش حذف شود؟')">@csrf @method('DELETE')<button class="btn sm danger" type="submit">حذف</button></form>
                                        @endunless
                                    </div>
                                </td>
                            </tr>
                        @empty
                            <tr><td colspan="8" class="empty">هنوز زبانی ثبت نشده است</td></tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>
    </div>
@endsection
