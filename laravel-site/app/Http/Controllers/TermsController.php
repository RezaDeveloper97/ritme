<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\StaticPages\StaticPageSeo;
use App\Domain\Settings\Contracts\SettingsRepository;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;

/**
 * /terms — pages/terms.blade.php. No design page (AUDIT §1/§8): the footer «شرایط استفاده» target, built in the
 * privacy-page layout (dark hero block, then the legal text). Copy: lang/fa/terms.php (a skeleton with «[…]» markers
 * for the legal team); support email + emergency number from settings; `terms.updated_at` → dateModified.
 */
final class TermsController
{
    public function __construct(
        private readonly StaticPageSeo $staticSeo,
        private readonly SettingsRepository $settings,
        private readonly ViewFactory $views,
    ) {}

    /** SeoManager and SchemaGraph are request-scoped: injected per call, not into the (route-cached) controller. */
    public function __invoke(SeoManager $seo, SchemaGraph $graph): View
    {
        $this->staticSeo->apply($seo, StaticPage::Terms);

        $updated = PrivacyController::date(self::text('terms.updated_at'));
        $graph->pageName(StaticPage::Terms->label())->dates(null, $updated?->toAtomString());

        $settings = $this->settings->all();

        return $this->views->make('pages.terms', [
            // Same layout as /privacy: the registry's default for terms is a light header.
            'headerVariant' => 'dark',
            'updatedAt' => $updated,
            'sections' => PrivacyController::legalSections(__('terms.doc.sections'), [
                ':support_email' => $settings->contact->supportEmail ?? self::text('terms.doc.support_missing'),
                ':emergency' => fa_digits($settings->general->emergencyNumber),
            ]),
        ]);
    }

    private static function text(string $key): string
    {
        $text = __($key);

        return is_string($text) ? $text : '';
    }
}
