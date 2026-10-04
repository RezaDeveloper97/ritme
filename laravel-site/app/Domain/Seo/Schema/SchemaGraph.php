<?php

declare(strict_types=1);

namespace App\Domain\Seo\Schema;

use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\Data\PersonData;
use App\Domain\Seo\Schema\Enums\WebPageType;

/**
 * The page's JSON-LD `@graph`, collected through the request (scoped; see SeoServiceProvider) and rendered once as
 * a single `<script type="application/ld+json">` by `<x-seo.head/>` (via PageGraph).
 *
 * Organization, WebSite, WebPage and BreadcrumbList are added automatically at render time; pages describe
 * themselves and add their own nodes — before the head renders (controller, or the page view's sections/slot):
 *
 *     $graph->pageType(WebPageType::AboutPage)
 *         ->breadcrumbs(new BreadcrumbItem('خانه', route('home')), new BreadcrumbItem('درباره ما'))
 *         ->add(FaqPageNode::make($url, $faqItems));
 *
 * Nodes are keyed by `@id`: adding a node with an existing `@id` replaces the given properties (later wins).
 */
final class SchemaGraph
{
    /** @var array<string, array<string, mixed>> */
    private array $nodes = [];

    private WebPageType $pageType = WebPageType::WebPage;

    private ?string $pageName = null;

    /** @var list<BreadcrumbItem>|null */
    private ?array $breadcrumbs = null;

    private ?PersonData $reviewedBy = null;

    private ?string $lastReviewed = null;

    private ?string $datePublished = null;

    private ?string $dateModified = null;

    private ?string $searchUrlTemplate = null;

    private int $blankNodes = 0;

    /**
     * @param  array<string, mixed>  $node
     */
    public function add(array $node): self
    {
        $key = $this->key($node);
        $this->nodes[$key] = array_replace($this->nodes[$key] ?? [], $node);

        return $this;
    }

    /**
     * Default nodes go first; properties already set by the page win.
     *
     * @param  list<array<string, mixed>>  $nodes
     */
    public function addDefaults(array $nodes): self
    {
        $defaults = [];
        foreach ($nodes as $node) {
            $key = $this->key($node);
            $defaults[$key] = array_replace($node, $this->nodes[$key] ?? []);
        }
        $this->nodes = $defaults + $this->nodes;

        return $this;
    }

    /**
     * Merges properties into an existing node, addressed by its full `@id` or by its fragment (`#webpage`).
     *
     * @param  array<string, mixed>  $properties
     */
    public function merge(string $idOrFragment, array $properties): self
    {
        foreach ($this->nodes as $key => $node) {
            $id = $node['@id'] ?? null;
            if (is_string($id) && ($id === $idOrFragment || (str_starts_with($idOrFragment, '#') && str_ends_with($id, $idOrFragment)))) {
                $this->nodes[$key] = array_replace($node, $properties);
            }
        }

        return $this;
    }

    public function remove(string $id): self
    {
        unset($this->nodes[$id]);

        return $this;
    }

    public function has(string $id): bool
    {
        return isset($this->nodes[$id]);
    }

    /**
     * @return array<string, mixed>|null
     */
    public function get(string $id): ?array
    {
        return $this->nodes[$id] ?? null;
    }

    public function pageType(WebPageType $type): self
    {
        $this->pageType = $type;

        return $this;
    }

    /**
     * Short page name for the last breadcrumb when no explicit trail is given (defaults to the title without the
     * site template).
     */
    public function pageName(string $name): self
    {
        $this->pageName = $name;

        return $this;
    }

    public function breadcrumbs(BreadcrumbItem ...$items): self
    {
        $this->breadcrumbs = array_values($items);

        return $this;
    }

    /**
     * YMYL hook: the (medical) expert who reviewed the page content, and when (ISO date).
     */
    public function reviewedBy(PersonData $reviewer, ?string $lastReviewed = null): self
    {
        $this->reviewedBy = $reviewer;
        $this->lastReviewed = $lastReviewed;

        return $this;
    }

    public function dates(?string $published, ?string $modified = null): self
    {
        $this->datePublished = $published;
        $this->dateModified = $modified;

        return $this;
    }

    /**
     * Turns on the WebSite SearchAction (L4-04). Must contain `{search_term_string}`.
     */
    public function searchUrlTemplate(?string $template): self
    {
        $this->searchUrlTemplate = $template;

        return $this;
    }

    public function getPageType(): WebPageType
    {
        return $this->pageType;
    }

    public function getPageName(): ?string
    {
        return $this->pageName;
    }

    /**
     * @return list<BreadcrumbItem>|null
     */
    public function getBreadcrumbs(): ?array
    {
        return $this->breadcrumbs;
    }

    public function getReviewedBy(): ?PersonData
    {
        return $this->reviewedBy;
    }

    public function getLastReviewed(): ?string
    {
        return $this->lastReviewed;
    }

    public function getDatePublished(): ?string
    {
        return $this->datePublished;
    }

    public function getDateModified(): ?string
    {
        return $this->dateModified;
    }

    public function getSearchUrlTemplate(): ?string
    {
        return $this->searchUrlTemplate;
    }

    /**
     * @return list<array<string, mixed>>
     */
    public function nodes(): array
    {
        $nodes = [];
        foreach ($this->nodes as $node) {
            /** @var array<string, mixed> $clean */
            $clean = Node::clean($node);
            if ($clean !== []) {
                $nodes[] = $clean;
            }
        }

        return $nodes;
    }

    /**
     * @return array{'@context': string, '@graph': list<array<string, mixed>>}
     */
    public function toArray(): array
    {
        return ['@context' => 'https://schema.org', '@graph' => $this->nodes()];
    }

    /**
     * Compact JSON, Persian unescaped, `<`/`>` hex-escaped so the payload can never close its <script>.
     */
    public function toJson(): string
    {
        return json_encode($this->toArray(), JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES | JSON_HEX_TAG | JSON_THROW_ON_ERROR);
    }

    public function toScript(): string
    {
        return '<script type="application/ld+json">'.$this->toJson().'</script>';
    }

    /**
     * @param  array<string, mixed>  $node
     */
    private function key(array $node): string
    {
        $id = $node['@id'] ?? null;

        return is_string($id) && $id !== '' ? $id : '_:b'.$this->blankNodes++;
    }
}
