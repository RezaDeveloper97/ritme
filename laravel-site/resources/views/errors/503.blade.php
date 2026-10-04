{{-- HTTP 503 (maintenance mode): standalone (errors.standalone), renders without the database, settings or cache. --}}
@include('errors.standalone', [
    'code' => 503,
    'title' => 'ریتمی در حال به‌روزرسانی است',
    'message' => 'به‌زودی برمی‌گردیم. چند دقیقه دیگر دوباره سر بزن.',
])
