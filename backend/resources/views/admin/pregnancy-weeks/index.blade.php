@extends('admin.layouts.app')

@section('title', 'محتوای هفتگی بارداری')

@section('content')
    <div class="card">
        <div class="card-head">
            <h2>هفته‌های بارداری (۱ تا ۴۰)</h2>
            <div class="spacer"></div>
            <span class="muted">سبز = محتوا دارد</span>
        </div>
        <div class="card-body">
            <div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(84px,1fr));gap:10px">
                @foreach ($weeks as $w)
                    @if ($w['id'])
                        <a class="btn" style="flex-direction:column;padding:12px 0;background:var(--success-soft);border-color:var(--success-soft);color:var(--success)"
                           href="{{ route('admin.pregnancy-weeks.edit', $w['id']) }}">
                            <strong style="font-size:16px">{{ $w['week'] }}</strong>
                            <span style="font-size:11px">ویرایش</span>
                        </a>
                    @else
                        <a class="btn" style="flex-direction:column;padding:12px 0"
                           href="{{ route('admin.pregnancy-weeks.create', ['week' => $w['week']]) }}">
                            <strong style="font-size:16px">{{ $w['week'] }}</strong>
                            <span style="font-size:11px" class="muted">افزودن</span>
                        </a>
                    @endif
                @endforeach
            </div>
        </div>
    </div>
@endsection
