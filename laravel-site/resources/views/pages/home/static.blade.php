{{--
    The cached middle of the home page: stages → #features → steps → partner → privacy. Copy, routes, the «صفحه‌ها»
    switches and the (static) site photos only; HomeController renders it once and caches the HTML in the `pages`
    namespace (key carries the photo ids). Never put per-request data or `priority` images in here.
    Data: $photos array<string, MediaData|null> keyed by SitePhoto value.
--}}
@include('pages.home.stages')
@include('pages.home.features')
@include('pages.home.steps')
@include('pages.home.partner')
@include('pages.home.privacy')
