<?php

declare(strict_types=1);

namespace App\Domain\Faq;

use App\Domain\Faq\Contracts\FaqRepository;
use App\Domain\Faq\Data\FaqGroupData;
use App\Domain\Faq\Data\FaqItemData;
use App\Domain\Seo\Schema\Nodes\FaqPageNode;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;
use Illuminate\Contracts\Container\Container;

/**
 * The FAQ blocks of the current page and their structured data. Every group a page *renders* goes through here, so
 * the request's SchemaGraph carries exactly one FAQPage node (`{canonical}#faq`) holding all visible questions —
 * never a question that is not on the page. Call it before the head renders (controller, or a view composer of the
 * page view; see FaqServiceProvider).
 */
final class PageFaq
{
    /**
     * The request-scoped SchemaGraph and SeoManager are resolved when used, not injected: controllers (and so this
     * service) can outlive a request in long-running workers and tests, the scoped graph cannot.
     */
    public function __construct(
        private readonly FaqRepository $faq,
        private readonly Container $container,
    ) {}

    /**
     * The published group, registered as visible; null when it does not exist or has no published item (the page
     * then renders no block at all).
     */
    public function group(string $slug): ?FaqGroupData
    {
        $group = $this->faq->group($slug);
        if ($group === null || $group->isEmpty()) {
            return null;
        }

        $this->show($group);

        return $group;
    }

    /**
     * Adds the groups' items to the page's FAQPage node (created on first use, merged afterwards; a question already
     * present is not repeated).
     */
    public function show(FaqGroupData ...$groups): void
    {
        $items = [];
        foreach ($groups as $group) {
            foreach ($group->items as $item) {
                $items[] = $item;
            }
        }
        if ($items === []) {
            return;
        }

        $graph = $this->container->make(SchemaGraph::class);
        $url = $this->container->make(SeoManager::class)->resolve()->canonical;
        $node = FaqPageNode::make($url, array_map(static fn (FaqItemData $item) => $item->toSchema(), $items));

        $existing = $graph->get(SchemaIds::faq($url))['mainEntity'] ?? [];
        if (is_array($existing) && $existing !== []) {
            $seen = [];
            $merged = [];
            foreach ([...$existing, ...(array) $node['mainEntity']] as $question) {
                $name = is_array($question) ? (string) ($question['name'] ?? '') : '';
                if ($name !== '' && ! isset($seen[$name])) {
                    $seen[$name] = true;
                    $merged[] = $question;
                }
            }
            $node['mainEntity'] = $merged;
        }

        $graph->add($node);
    }
}
