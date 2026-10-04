<?php

declare(strict_types=1);

namespace App\View\Components\Ui;

use App\Domain\Content\Enums\StaticPage;
use App\Domain\Content\SiteNavigation;
use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\SchemaGraph;
use Illuminate\Contracts\View\View;
use Illuminate\Routing\Router;
use Illuminate\View\Component;

/**
 * <x-ui.breadcrumbs :items="[...BreadcrumbItem]" /> — the visible trail AND the BreadcrumbList JSON-LD: the same
 * items are registered on the request's SchemaGraph when the component is constructed.
 *
 * Without :items the trail comes from the StaticPage registry for the current route. The first item (home) is
 * kept in the JSON-LD but hidden visually, as in the design (`:show-home="true"` shows it).
 *
 * Must render BEFORE `<x-seo.head/>` prints the graph — inside a page's `@section` (sections of a child view are
 * rendered before its layout), never in the layout after the head.
 */
final class Breadcrumbs extends Component
{
    /** @var list<BreadcrumbItem> */
    public readonly array $trail;

    /**
     * @param  array<int, BreadcrumbItem>|null  $items
     */
    public function __construct(
        ?array $items = null,
        public readonly bool $showHome = false,
        ?SchemaGraph $graph = null,
    ) {
        $items ??= self::fromRegistry();
        $this->trail = array_values($items);

        if (count($this->trail) > 1) {
            ($graph ?? app(SchemaGraph::class))->breadcrumbs(...$this->trail);
        }
    }

    /**
     * @return list<BreadcrumbItem>
     */
    public function visible(): array
    {
        return $this->showHome ? $this->trail : array_slice($this->trail, 1);
    }

    public function shouldRender(): bool
    {
        return $this->visible() !== [];
    }

    public function render(): View
    {
        return view('components.ui.breadcrumbs');
    }

    /**
     * @return list<BreadcrumbItem>
     */
    private static function fromRegistry(): array
    {
        $page = StaticPage::forRoute(app(Router::class)->currentRouteName());

        return $page === null ? [] : app(SiteNavigation::class)->breadcrumbs($page);
    }
}
