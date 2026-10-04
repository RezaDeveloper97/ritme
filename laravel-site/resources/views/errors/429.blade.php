{{-- HTTP 429 on the site layout (errors/shell: noindex, helpful links). --}}
@include('errors.shell', [
    'code' => 429,
    'title' => 'درخواست‌ها کمی زیاد شد',
    'message' => 'در مدت کوتاهی درخواست‌های زیادی رسید. چند دقیقه صبر کن و دوباره امتحان کن.',
])
