@extends('admin.layouts.app')

@section('title', 'محتوای فازهای چرخه')

@section('content')
    <div class="card">
        <div class="card-head">
            {{-- Count is derived, not hard-coded: the list follows CycleSubphase::options(). --}}
            <h2>فازهای چرخه ({{ count($phases) }} فاز)</h2>
            <div class="spacer"></div>
            <span class="muted">سبز = محتوا دارد</span>
        </div>
        <div class="card-body">
            <div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(150px,1fr));gap:10px">
                @foreach ($phases as $p)
                    @if ($p['id'])
                        <a class="btn" style="flex-direction:column;padding:14px 8px;background:var(--success-soft);border-color:var(--success-soft);color:var(--success)"
                           href="{{ route('admin.phase-contents.edit', $p['id']) }}">
                            <strong style="font-size:14px">{{ $p['label'] }}</strong>
                            <span style="font-size:11px">ویرایش</span>
                        </a>
                    @else
                        <a class="btn" style="flex-direction:column;padding:14px 8px"
                           href="{{ route('admin.phase-contents.create', ['phase' => $p['value']]) }}">
                            <strong style="font-size:14px">{{ $p['label'] }}</strong>
                            <span style="font-size:11px" class="muted">افزودن</span>
                        </a>
                    @endif
                @endforeach
            </div>
        </div>
    </div>
@endsection
