<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Faq\Contracts\FaqRepository;
use App\Domain\Faq\PageFaq;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\SeoManager;
use App\Domain\Settings\Contracts\SettingsRepository;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;

/**
 * /faq (design/html/faq.html) — pages/faq.blade.php: the listed FAQ groups (cached FaqRepository, `faq` ns) as
 * categories with an anchor side nav and an in-page filter, then the app CTA.
 */
final class FaqController
{
    /** Design title «سؤالات متداول — ریتمی» is 21 chars (seo:audit wants 30–60); this keeps it and names the brand once. */
    public const TITLE = 'سؤالات متداول — جواب سؤال‌های رایج درباره ریتمی';

    /** The design's description is the generic site line; this one says what the page answers. */
    public const DESCRIPTION = 'جواب سؤال‌های رایج درباره ریتمی: شروع کار، دقت پیش‌بینی‌ها و سلامت، حریم خصوصی و داده، پرداخت و اشتراک، خدمات و فروشگاه.';

    public function __construct(
        private readonly FaqRepository $faq,
        private readonly PageFaq $pageFaq,
        private readonly SeoMetaRepository $meta,
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
        $meta = $this->meta->forRoute(StaticPage::Faq->routeName());
        if (($meta->title ?? '') === '') {
            $seo->rawTitle(self::TITLE);
        }
        if (($meta->description ?? '') === '') {
            $seo->description(self::DESCRIPTION);
        }
        $this->pageFaq->show(...$groups);

        return $this->views->make('pages.faq', [
            'groups' => $groups,
            'appLinks' => $settings->appLinks,
            'qrUrl' => $settings->appLinks->webApp ?? $this->navigation->url(StaticPage::Faq, 'download', absolute: true),
        ]);
    }
}
