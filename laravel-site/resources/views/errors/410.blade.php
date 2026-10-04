{{-- HTTP 410 on the site layout (errors/shell: noindex, helpful links). --}}
@include('errors.shell', [
    'code' => 410,
    'title' => 'این صفحه دیگر وجود ندارد',
    'message' => 'این محتوا برای همیشه برداشته شده است. صفحه‌های تازه ریتمی را از پیوندهای زیر ببین.',
])
