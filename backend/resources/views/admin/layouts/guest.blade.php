<!DOCTYPE html>
<html lang="fa" dir="rtl">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>@yield('title', 'ورود') — پنل مدیریت ریتمه</title>
    <link rel="stylesheet" href="{{ route('admin.style') }}">
</head>
<body>
<div class="login-wrap">
    <div class="login-card">
        @yield('content')
    </div>
</div>
</body>
</html>
