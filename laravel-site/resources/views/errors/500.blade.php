{{-- HTTP 500: standalone (errors.standalone), renders without the database, settings or cache. --}}
@include('errors.standalone', [
    'code' => 500,
    'title' => 'مشکلی پیش آمد',
    'message' => 'خطایی در سرور رخ داد و ما در جریانش هستیم. کمی بعد دوباره امتحان کن.',
])
