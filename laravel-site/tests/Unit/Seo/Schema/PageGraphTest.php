<?php

declare(strict_types=1);

use App\Domain\Seo\Schema\Data\BreadcrumbItem;
use App\Domain\Seo\Schema\Data\FaqItem;
use App\Domain\Seo\Schema\Data\PersonData;
use App\Domain\Seo\Schema\Enums\WebPageType;
use App\Domain\Seo\Schema\Nodes\FaqPageNode;
use App\Domain\Seo\Schema\Nodes\MobileApplicationNode;
use App\Domain\Seo\Schema\SchemaGraph;
use Tests\Unit\Seo\Schema\SchemaFixtures;

it('adds Organization, WebSite, WebPage and BreadcrumbList with stable @ids', function (): void {
    $graph = new SchemaGraph;
    SchemaFixtures::pageGraph($graph)->build(SchemaFixtures::head());
    $nodes = SchemaFixtures::byId($graph);

    expect(array_keys($nodes))->toBe([
        'https://ritme.test/#organization',
        'https://ritme.test/#website',
        'https://ritme.test/cycle#webpage',
        'https://ritme.test/cycle#breadcrumb',
    ])
        ->and($nodes['https://ritme.test/cycle#webpage'])->toMatchArray([
            '@type' => 'WebPage', 'name' => 'پیگیری چرخه — ریتمی', 'description' => 'چرخه‌ات را بشناس.',
            'breadcrumb' => ['@id' => 'https://ritme.test/cycle#breadcrumb'],
        ])
        ->and($nodes['https://ritme.test/cycle#webpage']['primaryImageOfPage']['@id'])->toBe('https://ritme.test/cycle#primaryimage')
        ->and($nodes['https://ritme.test/cycle#breadcrumb']['itemListElement'])->toBe([
            ['@type' => 'ListItem', 'position' => 1, 'name' => 'خانه', 'item' => 'https://ritme.test/'],
            ['@type' => 'ListItem', 'position' => 2, 'name' => 'پیگیری چرخه', 'item' => 'https://ritme.test/cycle'],
        ]);
});

it('skips the breadcrumb on the home page and marks the organization as its subject', function (): void {
    $graph = new SchemaGraph;
    SchemaFixtures::pageGraph($graph)->build(SchemaFixtures::head('https://ritme.test', 'ریتمی — همراه سلامت زنان'));
    $nodes = SchemaFixtures::byId($graph);

    expect($nodes)->not->toHaveKey('https://ritme.test#breadcrumb')
        ->and($nodes['https://ritme.test#webpage'])->not->toHaveKey('breadcrumb')
        ->and($nodes['https://ritme.test#webpage']['about'])->toBe(['@id' => 'https://ritme.test/#organization']);
});

it('uses what the page declared: type, trail, review hook, extra nodes, overrides', function (): void {
    $url = 'https://ritme.test/faq';
    $graph = (new SchemaGraph)
        ->pageType(WebPageType::FaqPage)
        ->breadcrumbs(new BreadcrumbItem('خانه', 'https://ritme.test/'), new BreadcrumbItem('سؤال‌های پرتکرار'))
        ->reviewedBy(new PersonData('دکتر نمونه'), '2026-09-01')
        ->add(FaqPageNode::make($url, [new FaqItem('سؤال؟', 'پاسخ.')]));

    SchemaFixtures::pageGraph($graph)->build(SchemaFixtures::head($url, 'سؤال‌ها — ریتمی', [
        '#webpage' => ['about' => ['@id' => 'https://ritme.test/#app']],
        '@graph' => [MobileApplicationNode::make(SchemaFixtures::settings(), SchemaFixtures::SITE)],
    ]));
    $nodes = SchemaFixtures::byId($graph);

    expect($nodes["{$url}#webpage"])->toMatchArray(['@type' => 'FAQPage', 'lastReviewed' => '2026-09-01', 'about' => ['@id' => 'https://ritme.test/#app']])
        ->and($nodes["{$url}#breadcrumb"]['itemListElement'][1])->toBe(['@type' => 'ListItem', 'position' => 2, 'name' => 'سؤال‌های پرتکرار', 'item' => $url])
        ->and($nodes)->toHaveKeys(["{$url}#faq", 'https://ritme.test/#app']);
});

it('matches the sample graph snapshot', function (): void {
    $graph = (new SchemaGraph)->pageName('درباره ما')->pageType(WebPageType::AboutPage);
    SchemaFixtures::pageGraph($graph)->build(SchemaFixtures::head('https://ritme.test/about', 'درباره ما — ریتمی'));

    $snapshot = __DIR__.'/snapshots/sample-graph.json';
    $actual = json_encode($graph->toArray(), JSON_PRETTY_PRINT | JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES | JSON_THROW_ON_ERROR)."\n";

    expect($actual)->toBe((string) file_get_contents($snapshot));
});
