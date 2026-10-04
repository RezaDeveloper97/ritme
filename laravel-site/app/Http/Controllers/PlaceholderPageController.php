<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\SeoManager;
use Illuminate\Contracts\View\View;
use Illuminate\Routing\Router;

/**
 * Stand-in for every page route until its page task (L3-*, L4-*, L5-*, L6-*) swaps in the real controller in
 * routes/web.php. Renders the site layout with one h1 and is always noindex, so an empty page is never indexed.
 * Route parameters (slugs, codes) are ignored on purpose.
 */
final class PlaceholderPageController
{
    /** Parameterised routes (not StaticPage cases) => [h1, list page they belong to]. */
    private const DETAIL_PAGES = [
        'blog.show' => ['مقاله', StaticPage::Blog],
        'directory.place' => ['معرفی مجموعه', StaticPage::Directory],
        'directory.booked' => ['رزرو ثبت شد', StaticPage::Directory],
        'shop.category' => ['دسته فروشگاه', StaticPage::Shop],
        'shop.product' => ['محصول', StaticPage::Shop],
        'shop.order' => ['سفارش ثبت شد', StaticPage::Shop],
    ];

    public function __invoke(Router $router, SeoManager $seo, SiteNavigation $navigation): View
    {
        $routeName = (string) $router->currentRouteName();
        $page = StaticPage::forRoute($routeName);

        if ($page !== null) {
            $title = $page->label();
            $breadcrumbs = $navigation->breadcrumbs($page);
        } else {
            [$title, $parent] = self::DETAIL_PAGES[$routeName] ?? ['ریتمی', StaticPage::Home];
            $breadcrumbs = [...$navigation->breadcrumbs($parent), new BreadcrumbItem($title)];
            $breadcrumbs[count($breadcrumbs) - 2] = new BreadcrumbItem($parent->label(), $navigation->url($parent, absolute: true));
        }

        $seo->title($title)->noindex();

        return view('errors.shell', [
            'title' => $title,
            'message' => 'این صفحه در حال آماده شدن است. تا آن موقع می‌توانی از صفحه‌های دیگر ریتمی دیدن کنی.',
            'breadcrumbs' => $breadcrumbs,
            'appCta' => false,
        ]);
    }
}
