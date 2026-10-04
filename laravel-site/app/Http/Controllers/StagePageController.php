<?php

declare(strict_types=1);

namespace App\Http\Controllers;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\Stages\Data\StagePageData;
use App\Domain\Content\Stages\StagePageBuilder;
use App\Domain\Content\Stages\StageRegistry;
use App\Domain\Seo\Contracts\SeoMetaRepository;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\Nodes\FaqPageNode;
use App\Domain\Seo\Schema\Nodes\MobileApplicationNode;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use App\Domain\Settings\Contracts\SettingsRepository;
use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Contracts\Container\Container;
use Illuminate\Contracts\View\View;
use Illuminate\Routing\Router;

/**
 * The six life-stage pages (`stage.*` routes) on one template, `pages/stages/show.blade.php`. The stage comes from
 * the route name; a stage whose StageDefinition does not exist yet keeps the noindex placeholder.
 */
final class StagePageController
{
    public function __construct(
        private readonly StageRegistry $stages,
        private readonly StagePageBuilder $builder,
        private readonly SeoManager $seo,
        private readonly SeoMetaRepository $meta,
        private readonly SchemaGraph $graph,
        private readonly SettingsRepository $settings,
        private readonly Config $config,
    ) {}

    public function __invoke(Router $router, Container $container): View
    {
        $page = StaticPage::forRoute($router->currentRouteName());
        $definition = $this->stages->forPage($page);

        if ($page === null || $definition === null) {
            /** @var View */
            return $container->call(PlaceholderPageController::class);
        }

        $data = $this->builder->build($definition);
        $this->describe($page, $data);

        return view('pages.stages.show', ['page' => $data]);
    }

    /**
     * Title/description are the admin's (seo_meta of the route) when set, else the stage copy. JSON-LD: WebPage
     * (automatic), BreadcrumbList خانه › مرحله‌ها › <stage>, FAQPage for the visible FAQ and the app the CTA offers.
     */
    private function describe(StaticPage $page, StagePageData $data): void
    {
        $meta = $this->meta->forRoute($page->routeName());
        if (($meta->title ?? '') === '') {
            $this->seo->title($data->seoTitle);
        }
        if (($meta->description ?? '') === '') {
            $this->seo->description($data->seoDescription);
        }

        // Same origin as the canonical / WebPage node (app.url), whatever host the request came in on.
        $siteUrl = SchemaIds::root((string) $this->config->get('app.url'));
        $canonical = $this->seo->resolve()->canonical;
        $this->graph->pageName($data->name)->breadcrumbs(
            new BreadcrumbItem(StaticPage::Home->label(), $siteUrl),
            new BreadcrumbItem($data->stagesLabel, $siteUrl.'#stages'),
            new BreadcrumbItem($data->name, $canonical),
        );

        if ($data->faq !== []) {
            $this->graph->add(FaqPageNode::make($canonical, $data->faq));
        }

        $this->graph->add(MobileApplicationNode::make($this->settings->all(), $siteUrl));
    }
}
