<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema;

use App\Domain\Seo\Contracts\OgImageResolver;
use App\Domain\Seo\Data\SeoHead;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\Nodes\BreadcrumbListNode;
use App\Domain\Seo\Schema\Nodes\OrganizationNode;
use App\Domain\Seo\Schema\Nodes\WebPageNode;
use App\Domain\Seo\Schema\Nodes\WebSiteNode;
use App\Domain\Settings\Contracts\SettingsRepository;
use App\Domain\Settings\Data\SeoDefaults;
use Illuminate\Contracts\Config\Repository as Config;

/**
 * Completes the request's SchemaGraph with the site-wide nodes (Organization, WebSite), the page node (WebPage …)
 * and the BreadcrumbList, applies the admin `schema_overrides` of seo_meta, and renders the single JSON-LD script.
 *
 * schema_overrides format: `{"#webpage": {"about": …}, "@graph": [{…extra node…}]}` — keys are an `@id` or a
 * fragment whose node gets the properties merged (shallow); `@graph` appends whole nodes.
 */
final class PageGraph
{
    public const HOME_LABEL = 'خانه';

    public function __construct(
        private readonly SchemaGraph $graph,
        private readonly SettingsRepository $settings,
        private readonly OgImageResolver $images,
        private readonly Config $config,
    ) {}

    public function build(SeoHead $head): SchemaGraph
    {
        $settings = $this->settings->all();
        $siteUrl = SchemaIds::root((string) $this->config->get('app.url'));
        $url = $head->canonical;
        $logo = $settings->organization->logoMediaId !== null
            ? $this->images->resolve($settings->organization->logoMediaId, $settings->general->siteName)
            : null;

        $trail = $this->graph->getBreadcrumbs() ?? $this->defaultTrail($url, $siteUrl, $head->title, $settings->seo);
        $hasBreadcrumb = count($trail) > 1;

        $defaults = [
            OrganizationNode::make($settings, $siteUrl, $logo),
            WebSiteNode::make($settings, $siteUrl, $this->graph->getSearchUrlTemplate()),
            WebPageNode::make(
                url: $url,
                siteUrl: $siteUrl,
                name: $head->title,
                description: $head->description,
                type: $this->graph->getPageType(),
                image: $head->image,
                hasBreadcrumb: $hasBreadcrumb,
                datePublished: $this->graph->getDatePublished(),
                dateModified: $this->graph->getDateModified(),
                reviewedBy: $this->graph->getReviewedBy(),
                lastReviewed: $this->graph->getLastReviewed(),
            ),
        ];
        if ($hasBreadcrumb) {
            $defaults[] = BreadcrumbListNode::make($url, $trail);
        }
        $this->graph->addDefaults($defaults);

        foreach ($head->schemaOverrides as $key => $value) {
            if ($key === '@graph' && is_array($value)) {
                foreach ($value as $node) {
                    if (is_array($node) && $node !== []) {
                        /** @var array<string, mixed> $node */
                        $this->graph->add($node);
                    }
                }
            } elseif (is_array($value)) {
                /** @var array<string, mixed> $value */
                $this->graph->merge($key, $value);
            }
        }

        return $this->graph;
    }

    public function script(SeoHead $head): string
    {
        return $this->build($head)->toScript();
    }

    /**
     * Home → current page; the home page itself has no trail.
     *
     * @return list<BreadcrumbItem>
     */
    private function defaultTrail(string $url, string $siteUrl, string $title, SeoDefaults $seo): array
    {
        if (rtrim($url, '/') === rtrim($siteUrl, '/')) {
            return [new BreadcrumbItem(self::HOME_LABEL, $siteUrl)];
        }

        return [
            new BreadcrumbItem(self::HOME_LABEL, $siteUrl),
            new BreadcrumbItem($this->graph->getPageName() ?? $this->stripTemplate($title, $seo->template()), $url),
        ];
    }

    private function stripTemplate(string $title, string $template): string
    {
        [$prefix, $suffix] = array_pad(explode('%s', $template, 2), 2, '');
        if ($prefix !== '' && str_starts_with($title, $prefix)) {
            $title = substr($title, strlen($prefix));
        }
        if ($suffix !== '' && str_ends_with($title, $suffix)) {
            $title = substr($title, 0, -strlen($suffix));
        }

        return trim($title) !== '' ? trim($title) : $title;
    }
}
