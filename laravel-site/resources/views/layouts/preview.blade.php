{{--
    Shell preview for fidelity checks (L1-02): a blank page on the layout. Route `preview.layout` exists outside
    production only (routes/web.php). Dark: home is the active item and an empty hero of the design's hero height
    keeps the shared night/glow block the same size as on index.html. Light: no active item (as on faq.html).
--}}
@extends('layouts.app', [
    'headerVariant' => $variant,
    'navRoute' => $variant === 'dark' ? 'home' : 'faq',
    'appCta' => true,
])

@if ($variant === 'dark')
    @section('hero')
        <div class="h-[790px] max-lg:h-[1180px] max-sm:h-[1010px]"></div>
    @endsection
@endif

@section('content')
    <x-ui.section pad="lg">
        <x-ui.breadcrumbs :items="[
            new \App\Domain\Seo\Schema\Data\BreadcrumbItem('خانه', url('/')),
            new \App\Domain\Seo\Schema\Data\BreadcrumbItem('پیش‌نمایش قالب'),
        ]" :show-home="true"/>
        <h1 class="mt-6 text-d-xl">پیش‌نمایش قالب</h1>
        <div class="mt-6 flex flex-wrap items-center gap-3">
            <x-ui.button href="#main" size="lg" icon="download">دکمه اصلی</x-ui.button>
            <x-ui.button href="#main" variant="outline" size="lg">دکمه خطی</x-ui.button>
            <x-ui.button href="#main" variant="ghost" icon-end="arrow-left">بیشتر</x-ui.button>
            <x-ui.badge icon="shield-check">مدارک بررسی شد</x-ui.badge>
            <x-ui.badge tone="success">باز است</x-ui.badge>
        </div>
    </x-ui.section>
@endsection
