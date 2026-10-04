<?php

declare(strict_types=1);

namespace App\Http\Controllers\Pwa;

use App\Domain\Seo\SeoManager;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;

/**
 * `/offline` (L8-02): the page the service worker precaches and shows for a navigation that fails without a cached
 * copy. Static copy on the site layout, noindex, not in the sitemap (not a StaticPage).
 */
final class OfflineController
{
    public function __construct(
        private readonly SeoManager $seo,
        private readonly ViewFactory $views,
    ) {}

    public function __invoke(): View
    {
        $this->seo->title('اتصال به اینترنت برقرار نیست — ریتمی آفلاین')
            ->description('ریتمی بدون اینترنت هم کنارت است: صفحه‌هایی که قبلاً دیده‌ای در دسترس می‌مانند و با برگشت اتصال، بقیه هم باز می‌شوند.')
            ->noindex();

        return $this->views->make('pages.offline');
    }
}
