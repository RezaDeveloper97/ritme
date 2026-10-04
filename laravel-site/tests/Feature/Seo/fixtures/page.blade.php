<!doctype html>
<html lang="fa" dir="rtl">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<x-seo.head/>
</head>
<body>
<main id="main">
<h1>{{ $heading ?? 'پیگیری چرخه' }}</h1>
@if ($broken ?? false)
<h1>تیتر دوم</h1>
<img src="/media/a.webp" alt="نمونه">
<a href="#">ادامه</a>
@endif
</main>
</body>
</html>
