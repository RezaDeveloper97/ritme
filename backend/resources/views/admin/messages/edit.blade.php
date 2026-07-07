@extends('admin.layouts.app')

@section('title', 'ویرایش پیام')

@section('content')
    <div class="page-actions">
        <a class="btn ghost" href="{{ route('admin.messages.index', ['group' => $message->group]) }}">→ بازگشت</a>
    </div>

    <div class="card">
        <div class="card-head">
            <h2>{{ $labels[$message->group] ?? $message->group }}</h2>
            <div class="spacer"></div>
            <span class="badge">کلید: <span dir="ltr">{{ $message->item_key }}</span></span>
            <span class="badge pink">{{ $message->locale === 'fa' ? 'فارسی' : 'English' }}</span>
        </div>
        <div class="card-body">
            <p class="muted" style="margin-top:0">در فیلدهای فهرستی (مثل باید‌ها/نبایدها)، هر مورد را در یک خط بنویسید.</p>

            <form method="POST" action="{{ route('admin.messages.update', $message) }}">
                @csrf
                @method('PUT')

                @foreach ($message->payload as $key => $value)
                    <div class="field">
                        <label for="f_{{ $key }}">{{ $key }} @if(is_array($value))<span class="hint">(فهرست — هر خط یک مورد)</span>@endif</label>
                        @if (is_array($value))
                            <textarea id="f_{{ $key }}" name="payload[{{ $key }}]" dir="{{ $message->locale === 'fa' ? 'rtl' : 'ltr' }}" rows="{{ max(3, count($value)) }}">{{ implode("\n", $value) }}</textarea>
                        @elseif (mb_strlen((string) $value) > 60)
                            <textarea id="f_{{ $key }}" name="payload[{{ $key }}]" dir="{{ $message->locale === 'fa' ? 'rtl' : 'ltr' }}">{{ $value }}</textarea>
                        @else
                            <input type="text" id="f_{{ $key }}" name="payload[{{ $key }}]" dir="{{ $message->locale === 'fa' ? 'rtl' : 'ltr' }}" value="{{ $value }}">
                        @endif
                    </div>
                @endforeach

                <div class="form-actions">
                    <button class="btn primary" type="submit">ذخیره</button>
                </div>
            </form>
        </div>
    </div>
@endsection
