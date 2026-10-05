<?php

declare(strict_types=1);

use App\Domain\Content\Enums\StaticPage;
use Database\Seeders\SettingsSeeder;

/*
 * The router keeps one controller instance per route for the whole test (as Octane and the in-process SEO audit do).
 * Request-scoped SeoManager / SchemaGraph are method-injected, so a second visit of the same route — after another
 * page in between — renders its own <title>, description and a single JSON-LD graph again.
 */

beforeEach(function (): void {
    $this->seed(SettingsSeeder::class);
});

/**
 * @return array{title: string, description: string, graphs: int, breadcrumbs: int}
 */
function pageHead(string $html): array
{
    preg_match('#<title>(.*?)</title>#s', $html, $title);
    preg_match('#<meta name="description" content="([^"]*)"#', $html, $description);

    return [
        'title' => trim($title[1] ?? ''),
        'description' => $description[1] ?? '',
        'graphs' => substr_count($html, '"@graph"'),
        'breadcrumbs' => substr_count($html, '"@type":"BreadcrumbList"'),
    ];
}

it('renders the same head on repeat visits through a reused controller', function (): void {
    $pages = [
        StaticPage::Home, StaticPage::Cycle, StaticPage::Services, StaticPage::Plus, StaticPage::Tools, StaticPage::About,
        StaticPage::SocialResponsibility, StaticPage::Privacy, StaticPage::Terms, StaticPage::Faq, StaticPage::Contact,
        StaticPage::Blog, StaticPage::Directory, StaticPage::DirectoryBusiness, StaticPage::Shop,
    ];
    $visit = fn (StaticPage $page): array => pageHead((string) $this->withHeader('Cache-Control', 'no-cache')
        ->get(route($page->routeName()))->assertOk()->getContent());

    $first = [];
    foreach ($pages as $page) {
        $first[$page->value] = $visit($page);
        expect($first[$page->value]['title'])->not->toBe('')
            ->and($first[$page->value]['graphs'])->toBe(1);
    }
    foreach (array_reverse($pages) as $page) {
        expect($visit($page))->toBe($first[$page->value], $page->value);
    }
});
