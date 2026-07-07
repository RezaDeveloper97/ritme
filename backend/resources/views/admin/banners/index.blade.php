@extends('admin.layouts.app')

@section('title', 'بنرها')

@section('content')
    <div class="page-actions">
        <div class="spacer"></div>
        <a class="btn primary" href="{{ route('admin.banners.create') }}">+ بنر جدید</a>
    </div>

    <div class="card">
        <div class="card-body tight">
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr>
                            <th>#</th>
                            <th>تصویر</th>
                            <th>عنوان</th>
                            <th>جایگاه</th>
                            <th>لینک</th>
                            <th>بازه نمایش</th>
                            <th>وضعیت</th>
                            <th>ترتیب</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        @php $positionLabels = collect($positions)->keyBy->value; @endphp
                        @forelse ($banners as $banner)
                            <tr>
                                <td>{{ $banner->id }}</td>
                                <td>
                                    @if ($banner->image_url)
                                        <img src="{{ $banner->image_url }}" alt=""
                                             style="width:96px;height:48px;object-fit:cover;border-radius:8px;display:block">
                                    @else
                                        —
                                    @endif
                                </td>
                                <td class="wrap">{{ $banner->title['fa'] ?? '—' }}</td>
                                <td>{{ optional($positionLabels->get($banner->position))->label('fa') ?? $banner->position }}</td>
                                <td dir="ltr" style="max-width:180px" class="wrap">
                                    @if ($banner->link_url)
                                        <span class="badge {{ $banner->link_type === 'external' ? 'amber' : 'pink' }}">
                                            {{ $banner->link_type === 'external' ? 'خارجی' : 'داخلی' }}
                                        </span>
                                        <div class="muted" style="font-size:11px">{{ \Illuminate\Support\Str::limit($banner->link_url, 40) }}</div>
                                    @else
                                        —
                                    @endif
                                </td>
                                <td style="font-size:12px" class="wrap">
                                    {{ $banner->starts_at?->format('Y/m/d H:i') ?? '—' }}
                                    <br>تا {{ $banner->ends_at?->format('Y/m/d H:i') ?? '∞' }}
                                </td>
                                <td>
                                    @if ($banner->is_active)
                                        <span class="badge green">فعال</span>
                                    @else
                                        <span class="badge amber">غیرفعال</span>
                                    @endif
                                </td>
                                <td>{{ $banner->sort_order }}</td>
                                <td>
                                    <div class="btn-row">
                                        <a class="btn sm" href="{{ route('admin.banners.edit', $banner) }}">ویرایش</a>
                                        <form method="POST" action="{{ route('admin.banners.toggle', $banner) }}">
                                            @csrf
                                            <button class="btn sm" type="submit">{{ $banner->is_active ? 'غیرفعال' : 'فعال' }}</button>
                                        </form>
                                        <form method="POST" action="{{ route('admin.banners.destroy', $banner) }}" onsubmit="return confirm('حذف شود؟')">
                                            @csrf @method('DELETE')
                                            <button class="btn sm danger" type="submit">حذف</button>
                                        </form>
                                    </div>
                                </td>
                            </tr>
                        @empty
                            <tr><td colspan="9" class="empty">بنری ثبت نشده است</td></tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    {{ $banners->links('admin.partials.pagination') }}
@endsection
