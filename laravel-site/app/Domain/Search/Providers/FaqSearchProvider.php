<?php

declare(strict_types=1);

namespace App\Domain\Search\Providers;

use App\Domain\Faq\Contracts\FaqRepository;
use App\Domain\Search\Contracts\SearchProvider;
use App\Domain\Search\Data\SearchHit;
use App\Domain\Search\Support\SearchTerms;
use App\Support\Html\HtmlText;
use Illuminate\Contracts\Routing\UrlGenerator;
use Illuminate\Routing\Router;
use Illuminate\Support\Str;

/**
 * Published questions of the groups listed on /faq. The FAQ is small and already cached (`faq` namespace), so it is
 * filtered in PHP with the same normalisation as SQL providers. Links point at the question's group on /faq.
 */
final class FaqSearchProvider implements SearchProvider
{
    public function __construct(
        private readonly FaqRepository $faq,
        private readonly UrlGenerator $url,
        private readonly Router $router,
    ) {}

    public function key(): string
    {
        return 'faq';
    }

    public function label(): string
    {
        return (string) __('search.types.faq');
    }

    public function search(SearchTerms $terms, int $limit): array
    {
        $page = $this->router->has('faq') ? $this->url->route('faq') : $this->url->to('/faq');
        $hits = [];

        foreach ($this->faq->listed() as $group) {
            foreach ($group->items as $item) {
                $answer = HtmlText::plain($item->answer);
                if (! $terms->matches($item->question.' '.$answer)) {
                    continue;
                }

                $hits[] = new SearchHit(
                    type: $this->key(),
                    typeLabel: $this->label(),
                    title: $item->question,
                    url: $page.'#'.rawurlencode($group->slug),
                    snippet: $answer === '' ? null : Str::limit($answer, 180),
                    score: $terms->touches($item->question) ? 2 : 1,
                );
            }
        }

        usort($hits, static fn (SearchHit $a, SearchHit $b): int => $b->score <=> $a->score);

        return array_slice($hits, 0, max(1, $limit));
    }
}
