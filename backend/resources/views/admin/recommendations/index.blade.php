@extends('admin.layouts.app')

@section('title', 'توصیه‌های امروز')

@section('content')
    <div class="page-actions">
        <div class="spacer"></div>
        <a class="btn primary" href="{{ route('admin.recommendations.create') }}">+ توصیه جدید</a>
    </div>

    <div class="card">
        <div class="card-head">
            <h2>توصیه‌های امروز</h2>
            <div class="spacer"></div>
            <form method="GET" class="btn-row" style="gap:8px">
                <select name="phase" onchange="this.form.submit()">
                    <option value="">همه فازها</option>
                    <option value="general" @selected($phase === 'general')>بدون فاز (عمومی)</option>
                    @foreach ($phases as $p)
                        <option value="{{ $p['value'] }}" @selected($phase === $p['value'])>{{ $p['label'] }}</option>
                    @endforeach
                </select>
                <select name="type" onchange="this.form.submit()">
                    <option value="">همه دسته‌ها</option>
                    @foreach ($types as $t)
                        <option value="{{ $t['value'] }}" @selected($type === $t['value'])>{{ $t['label'] }}</option>
                    @endforeach
                </select>
            </form>
        </div>
        <div class="card-body tight">
            <p class="hint" style="margin:0 0 10px">
                این توصیه‌ها در کارت «توصیه‌های امروز» صفحه‌ی خانه دیده می‌شوند. هر توصیه به یک فاز سیکل
                تعلق دارد و می‌تواند به زیرفازهای مشخص یا به یک علامت ثبت‌شده هم محدود شود.
                برای پنهان‌کردن یک توصیه آن را <strong>غیرفعال</strong> کنید؛ حذف فقط برای ردیف‌هایی است
                که دیگر لازم نیستند — اگر هیچ ردیفی در جدول نماند، برنامه به متن‌های پیش‌فرض داخل کد برمی‌گردد.
            </p>
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr><th>#</th><th>دسته</th><th>متن</th><th>فاز</th><th>زیرفازها</th><th>علامت</th><th>وضعیت</th><th>ترتیب</th><th></th></tr>
                    </thead>
                    <tbody>
                        @forelse ($recommendations as $item)
                            <tr>
                                <td>{{ $item->id }}</td>
                                <td>{{ \App\Enums\RecommendationType::labelFor($item->type) }}</td>
                                <td class="wrap" style="max-width:360px">{{ \Illuminate\Support\Str::limit($item->text['fa'] ?? $item->text['en'] ?? '—', 90) }}</td>
                                <td>{{ \App\Enums\CyclePhase::labelFor($item->cycle_phase) ?? 'همه فازها' }}</td>
                                <td class="wrap muted" style="max-width:200px">
                                    @forelse ($item->cycle_subphases ?? [] as $sub)
                                        <span class="badge">{{ \App\Enums\CycleSubphase::labelFor($sub) ?? $sub }}</span>
                                    @empty
                                        —
                                    @endforelse
                                </td>
                                <td>{{ \App\Enums\RecommendationTrigger::labelFor($item->symptom_trigger) ?? '—' }}</td>
                                <td>@if ($item->is_active)<span class="badge green">فعال</span>@else<span class="badge">غیرفعال</span>@endif</td>
                                <td>{{ $item->sort_order }}</td>
                                <td>
                                    <div class="btn-row">
                                        <a class="btn sm" href="{{ route('admin.recommendations.edit', $item) }}">ویرایش</a>
                                        <form method="POST" action="{{ route('admin.recommendations.toggle', $item) }}">@csrf<button class="btn sm" type="submit">{{ $item->is_active ? 'غیرفعال' : 'فعال' }}</button></form>
                                        <form method="POST" action="{{ route('admin.recommendations.destroy', $item) }}" onsubmit="return confirm('حذف شود؟')">@csrf @method('DELETE')<button class="btn sm danger" type="submit">حذف</button></form>
                                    </div>
                                </td>
                            </tr>
                        @empty
                            <tr><td colspan="9" class="empty">موردی ثبت نشده است</td></tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    {{ $recommendations->links('admin.partials.pagination') }}
@endsection
