@extends('admin.layouts.app')

@section('title', 'مقالات')

@section('content')
    <div class="page-actions">
        <div class="spacer"></div>
        <a class="btn primary" href="{{ route('admin.articles.create') }}">+ مقاله جدید</a>
    </div>

    <div class="card">
        <div class="card-body tight">
            <div class="table-wrap">
                <table class="data">
                    <thead>
                        <tr>
                            <th>#</th>
                            <th>عنوان</th>
                            <th>فاز</th>
                            <th>دسته</th>
                            <th>وضعیت</th>
                            <th>ترتیب</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        @forelse ($articles as $article)
                            <tr>
                                <td>{{ $article->id }}</td>
                                <td class="wrap">{{ $article->title['fa'] ?? '—' }}</td>
                                {{-- Same labels as the phase picker; falls back to the raw key for legacy values. --}}
                                <td class="wrap">
                                    @forelse ($article->cycle_phases ?? [] as $phase)
                                        <span class="badge">{{ \App\Enums\CycleSubphase::labelFor($phase) ?? \App\Enums\CyclePhase::labelFor($phase) ?? $phase }}</span>
                                    @empty
                                        <span class="muted">همه فازها</span>
                                    @endforelse
                                </td>
                                <td>{{ $article->category ?: '—' }}</td>
                                <td>
                                    @if ($article->is_published)
                                        <span class="badge green">منتشر شده</span>
                                    @else
                                        <span class="badge amber">پیش‌نویس</span>
                                    @endif
                                </td>
                                <td>{{ $article->sort_order }}</td>
                                <td>
                                    <div class="btn-row">
                                        <a class="btn sm" href="{{ route('admin.articles.edit', $article) }}">ویرایش</a>
                                        <form method="POST" action="{{ route('admin.articles.toggle', $article) }}">
                                            @csrf
                                            <button class="btn sm" type="submit">{{ $article->is_published ? 'لغو انتشار' : 'انتشار' }}</button>
                                        </form>
                                        <form method="POST" action="{{ route('admin.articles.destroy', $article) }}" onsubmit="return confirm('حذف شود؟')">
                                            @csrf @method('DELETE')
                                            <button class="btn sm danger" type="submit">حذف</button>
                                        </form>
                                    </div>
                                </td>
                            </tr>
                        @empty
                            <tr><td colspan="7" class="empty">مقاله‌ای ثبت نشده است</td></tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    {{ $articles->links('admin.partials.pagination') }}
@endsection
