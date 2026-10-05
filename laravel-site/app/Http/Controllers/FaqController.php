<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Faq\Contracts\FaqRepository;
use App\Domain\Faq\PageFaq;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\StaticPages\StaticPageSeo;
use App\Domain\Settings\Contracts\SettingsRepository;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;

/**
 * /faq (design/html/faq.html) — pages/faq.blade.php: the listed FAQ groups (cached FaqRepository, `faq` ns) as
 * categories with an anchor side nav and an in-page filter, then the app CTA.
 */
final class FaqController
{
    public function __construct(
        private readonly FaqRepository $faq,
        private readonly PageFaq $pageFaq,
        private readonly StaticPageSeo $staticSeo,
        private readonly SettingsRepository $settings,
        private readonly SiteNavigation $navigation,
        private readonly ViewFactory $views,
    ) {}

    /** SeoManager is request-scoped: injected per call, not into the (route-cached) controller. */
    public function __invoke(SeoManager $seo): View
    {
        $settings = $this->settings->all();
        $groups = $this->faq->listed();

        // Title/description: the admin's (seo_meta of `faq`) when set. Breadcrumbs خانه › سؤالات متداول and the
        // WebPage node are automatic; one FAQPage node carries every question shown below.
        // Default title/description: StaticPageSeoDefaults::FAQ_TITLE / FAQ_DESCRIPTION.
        $this->staticSeo->apply($seo, StaticPage::Faq);
        $this->pageFaq->show(...$groups);

        return $this->views->make('pages.faq', [
            'groups' => $groups,
            'appLinks' => $settings->appLinks,
            'qrUrl' => $settings->appLinks->webApp ?? $this->navigation->url(StaticPage::Faq, 'download', absolute: true),
        ]);
    }
}
