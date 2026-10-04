<?php

declare(strict_types=1);

use App\Domain\Seo\Schema\Data\FaqItem;
use App\Domain\Seo\Schema\Nodes\FaqPageNode;
use App\Domain\Seo\Schema\SchemaGraph;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\Route;
use Tests\Feature\Seo\SeoFixtures;
use Tests\TestCase;

uses(TestCase::class, RefreshDatabase::class);

beforeEach(function (): void {
    SeoFixtures::boot();
});

it('renders exactly one JSON-LD @graph script in the layout head, including nodes the page added', function (): void {
    Route::get('/faq', function (SchemaGraph $graph) {
        $graph->add(FaqPageNode::make('https://ritme.test/faq', [new FaqItem('سؤال؟', 'پاسخ.')]));

        return view('seo-fixtures::page');
    })->name('fixture.faq');

    $html = (string) $this->get('/faq')->assertOk()->getContent();
    preg_match_all('#<script type="application/ld\+json">(.*?)</script>#s', $html, $scripts);
    $head = substr($html, 0, (int) strpos($html, '</head>'));

    expect($scripts[1])->toHaveCount(1)
        ->and($head)->toContain('application/ld+json');

    $graph = json_decode($scripts[1][0], true, 512, JSON_THROW_ON_ERROR);
    expect($graph['@context'])->toBe('https://schema.org')
        ->and(array_column($graph['@graph'], '@id'))->toBe([
            'https://ritme.test/#organization',
            'https://ritme.test/#website',
            'https://ritme.test/faq#webpage',
            'https://ritme.test/faq#breadcrumb',
            'https://ritme.test/faq#faq',
        ]);
});

it('starts every request with an empty graph', function (): void {
    Route::get('/a', function (SchemaGraph $graph) {
        $graph->add(['@type' => 'Thing', '@id' => 'https://ritme.test/#leak']);

        return view('seo-fixtures::page');
    });
    Route::get('/b', fn () => view('seo-fixtures::page'));

    $this->get('/a')->assertSee('#leak', false);
    $this->get('/b')->assertDontSee('#leak', false);
});
