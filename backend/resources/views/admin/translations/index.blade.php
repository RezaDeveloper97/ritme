@extends('admin.layouts.app')

@section('title', 'ترجمه‌های ' . $language->name)

@section('content')
    <div class="page-actions">
        <a class="btn ghost" href="{{ route('admin.languages.index') }}">→ بازگشت به زبان‌ها</a>
    </div>

    <div class="tabs">
        @foreach ($namespaces as $tab)
            <a class="tab {{ $tab === $namespace ? 'active' : '' }}"
               href="{{ route('admin.languages.translations.index', ['language' => $language, 'namespace' => $tab]) }}">
                {{ $tab }}
            </a>
        @endforeach
    </div>

    <div class="card">
        <div class="card-body">
            <p class="hint" style="margin:0 0 12px">
                رشته‌های رابط کاربری اپ برای زبان «{{ $language->name }}»، بخشِ «{{ $namespace }}».
                @if ($isDefaultLocale)
                    این زبانِ پیش‌فرض است؛ هر رشته‌ای که اینجا خالی بماند در اپ خالی دیده می‌شود.
                @else
                    ستون سمت مقابل، متنِ زبان پیش‌فرض ({{ $defaultName }}) است.
                    هر خانه‌ای که خالی بگذارید، در اپ همان متن پیش‌فرض نمایش داده می‌شود.
                @endif
                علامت‌هایی مثل <span dir="ltr">{'{'}count{'}'}</span> باید دست‌نخورده بمانند.
            </p>

            <form method="POST"
                  action="{{ route('admin.languages.translations.update', ['language' => $language, 'namespace' => $namespace]) }}">
                @csrf
                @method('PUT')

                <div class="table-wrap">
                    <table class="data">
                        <thead>
                            <tr>
                                <th style="width:28%">کلید</th>
                                @unless ($isDefaultLocale)<th style="width:32%">{{ $defaultName }}</th>@endunless
                                <th>{{ $language->name }}</th>
                            </tr>
                        </thead>
                        <tbody>
                            @forelse ($rows as $index => $row)
                                <tr>
                                    <td class="wrap" dir="ltr" style="font-size:12px;color:var(--muted)">{{ $row['key'] }}</td>
                                    @unless ($isDefaultLocale)
                                        <td class="wrap" dir="{{ app(\App\Services\Language\LanguageRegistry::class)->direction($defaultCode)->value }}">
                                            {{ $row['reference'] ?: '—' }}
                                        </td>
                                    @endunless
                                    <td>
                                        {{-- Message keys contain dots, which PHP's form parser would
                                             turn into nested input names, so each row carries its
                                             own key alongside the value instead. --}}
                                        <input type="hidden" name="rows[{{ $index }}][key]" value="{{ $row['key'] }}">
                                        <textarea name="rows[{{ $index }}][value]"
                                                  dir="{{ $language->direction->value }}"
                                                  rows="{{ mb_strlen($row['reference'] ?: $row['value']) > 90 ? 3 : 1 }}"
                                                  placeholder="{{ $isDefaultLocale ? '' : $row['reference'] }}">{{ $row['value'] }}</textarea>
                                    </td>
                                </tr>
                            @empty
                                <tr><td colspan="3" class="empty">برای این بخش رشته‌ای پیدا نشد</td></tr>
                            @endforelse
                        </tbody>
                    </table>
                </div>

                <div class="form-actions">
                    <button class="btn primary" type="submit">ذخیره‌ی بخش «{{ $namespace }}»</button>
                </div>
            </form>
        </div>
    </div>
@endsection
