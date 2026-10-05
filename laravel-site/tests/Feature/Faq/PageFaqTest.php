<?php

declare(strict_types=1);

use App\Domain\Faq\Data\FaqGroupData;
use App\Domain\Faq\Data\FaqItemData;
use App\Domain\Faq\PageFaq;
use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\Schema\SchemaIds;
use App\Domain\Seo\SeoManager;

function faqGroup(int $id, string ...$questions): FaqGroupData
{
    $items = [];
    foreach ($questions as $i => $question) {
        $items[] = new FaqItemData($id * 100 + $i, $question, "<p>جواب {$question}</p>");
    }

    return new FaqGroupData($id, "group-{$id}", "گروه {$id}", true, $items);
}

it('merges blocks shown on one page into a single FAQPage node without repeating a question', function (): void {
    $faq = app(PageFaq::class);

    $faq->show(faqGroup(1, 'سؤال یک', 'سؤال دو'));
    $faq->show(faqGroup(2, 'سؤال دو', 'سؤال سه'), faqGroup(3));

    $url = app(SeoManager::class)->resolve()->canonical;
    $node = app(SchemaGraph::class)->get(SchemaIds::faq($url));
    $names = array_map(static fn (array $q): string => $q['name'], $node['mainEntity'] ?? []);

    expect($names)->toBe(['سؤال یک', 'سؤال دو', 'سؤال سه']);
});

it('adds nothing for empty groups', function (): void {
    app(PageFaq::class)->show(faqGroup(1));

    expect(app(SchemaGraph::class)->get(SchemaIds::faq(app(SeoManager::class)->resolve()->canonical)))->toBeNull();
});
