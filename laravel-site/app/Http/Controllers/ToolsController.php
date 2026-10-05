<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Content\Tools\Data\JalaliDay;
use App\Domain\Content\Tools\ResultText;
use App\Domain\Content\Tools\ToolsCalculators;
use App\Domain\Content\Tools\ToolsSchema;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use App\Domain\Seo\StaticPages\StaticPageSeo;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Support\Jalali\JalaliDate;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\View\Factory as ViewFactory;
use Illuminate\Contracts\View\View;
use Illuminate\Http\Request;

/**
 * /tools (design/html/tools.html) — pages/tools.blade.php: due-date + fertility-window calculators, 3 checklists,
 * 4 guides, app CTA. Stable anchors: #due-date, #fertility, #hospital-bag, #sisemoni (footer + home link to them).
 *
 * The calculators run in the browser (calculators.js) and without JS: each card is a GET form, so
 * `/tools?calc=due&lmp=1405/02/12&cycle=28` renders the result here. Parameterised URLs are noindex with the
 * canonical on /tools, and the full-page cache skips them (it caches only parameterless / `page` URLs). The plain page
 * shows the example input of each card with its result and no date-dependent text, so it is safe to cache.
 */
final class ToolsController
{
    public function __construct(
        private readonly StaticPageSeo $staticSeo,
        private readonly SettingsRepository $settings,
        private readonly SiteNavigation $navigation,
        private readonly ToolsCalculators $calculators,
        private readonly ViewFactory $views,
        private readonly Config $config,
    ) {}

    /** SeoManager and SchemaGraph are request-scoped: injected per call, not into the (route-cached) controller. */
    public function __invoke(Request $request, SeoManager $seo, SchemaGraph $graph): View
    {
        $settings = $this->settings->all();
        $templates = self::list('tools.text');
        $text = new ResultText($templates);
        $submitted = ToolsCalculators::submitted($request->query('calc'));

        $this->describe($seo, $graph, $submitted);

        return $this->views->make('pages.tools', [
            'forms' => $this->calculators->forms(
                $request->query(),
                self::examples(),
                $text,
                JalaliDay::fromDate(JalaliDate::now()),
            ),
            'templates' => $templates,
            'errorsText' => self::list('tools.errors'),
            'appLinks' => $settings->appLinks,
            'qrUrl' => $settings->appLinks->webApp ?? $this->navigation->url(StaticPage::Tools, 'download', absolute: true),
        ]);
    }

    /**
     * Title/description: the admin's (seo_meta of `tools`) when set, else the design's. Submissions: noindex,
     * canonical → /tools. JSON-LD: one free WebApplication per calculator (breadcrumbs Home → ابزارها are automatic).
     */
    private function describe(SeoManager $seo, SchemaGraph $graph, bool $submitted): void
    {
        $pageUrl = $this->navigation->url(StaticPage::Tools, absolute: true);

        // The design title already carries the brand: no «%s — ریتمی» template.
        $this->staticSeo->apply($seo, StaticPage::Tools);
        if ($submitted) {
            $seo->noindex()->canonical($pageUrl);
        }

        $copy = [];
        foreach (['due', 'fert'] as $kind) {
            $copy[$kind] = [
                'name' => self::text("tools.calculators.{$kind}.title"),
                'description' => self::text("tools.calculators.{$kind}.schema"),
            ];
        }
        foreach (ToolsSchema::nodes($pageUrl, SchemaIds::root((string) $this->config->get('app.url')), $copy) as $node) {
            $graph->add($node);
        }
    }

    /**
     * @return array<string, array{lmp: string, cycle: string}>
     */
    private static function examples(): array
    {
        $examples = [];
        foreach (self::list('tools.calculators.examples') as $kind => $example) {
            $examples[(string) $kind] = [
                'lmp' => is_array($example) && is_string($example['lmp'] ?? null) ? $example['lmp'] : '',
                'cycle' => is_array($example) && is_string($example['cycle'] ?? null) ? $example['cycle'] : '',
            ];
        }

        return $examples;
    }

    /**
     * @return array<array-key, mixed>
     */
    private static function list(string $key): array
    {
        $value = __($key);

        return is_array($value) ? $value : [];
    }

    private static function text(string $key): string
    {
        $text = __($key);

        return is_string($text) ? $text : '';
    }
}
