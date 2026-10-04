{{-- HTTP 404 on the site layout (errors/shell: noindex, helpful links). --}}
@include('errors.shell', [
    'code' => 404,
    'title' => 'این صفحه پیدا نشد',
    'message' => 'نشانی‌ای که باز کردی وجود ندارد یا جابه‌جا شده است. از صفحه اصلی یا پیوندهای زیر ادامه بده.',
])
