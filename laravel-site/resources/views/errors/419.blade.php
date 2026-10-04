{{-- HTTP 419 on the site layout (errors/shell: noindex, helpful links). --}}
@include('errors.shell', [
    'code' => 419,
    'title' => 'زمان این صفحه تمام شده است',
    'message' => 'برای امنیت بیشتر، فرم‌ها پس از مدتی منقضی می‌شوند. صفحه را دوباره باز کن و دوباره امتحان کن.',
])
