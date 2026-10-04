<?php

declare(strict_types=1);

use App\Domain\Blog\Models\Category;
use App\Domain\Blog\Models\Post;
use App\Domain\Media\Models\Media;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Sitemap\SitemapEntryData;
use App\Domain\Seo\Sitemap\SitemapProvider;
use App\Domain\Seo\Sitemap\SitemapRegistry;
use App\Domain\Seo\Sitemap\Sitemaps;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\DB;
use Illuminate\Testing\TestResponse;

beforeEach(function (): void {
    config(['app.url' => 'https://ritme.ir']);
});

afterEach(fn () => Carbon::setTestNow());

/**
 * Parses a sitemap response and returns its namespaced root (fails the test on malformed XML).
 */
function sitemapXml(TestResponse $response): SimpleXMLElement
{
    $response->assertOk()->assertHeader('Content-Type', 'application/xml; charset=UTF-8');
    $previous = libxml_use_internal_errors(true);
    $xml = simplexml_load_string((string) $response->getContent());
    $errors = libxml_get_errors();
    libxml_clear_errors();
    libxml_use_internal_errors($previous);

    expect($errors)->toBe([])->and($xml)->toBeInstanceOf(SimpleXMLElement::class);
    assert($xml instanceof SimpleXMLElement);

    return $xml;
}

/**
 * @return list<string>
 */
function sitemapLocs(SimpleXMLElement $xml): array
{
    $xml->registerXPathNamespace('s', 'http://www.sitemaps.org/schemas/sitemap/0.9');

    return array_map('strval', $xml->xpath('//s:loc') ?: []);
}

function sitemapQueryCount(Closure $callback): int
{
    DB::flushQueryLog();
    DB::enableQueryLog();
    $callback();
    $count = count(DB::getQueryLog());
    DB::disableQueryLog();

    return $count;
}

function registerFakeSitemapProvider(SitemapProvider $provider): void
{
    app()->tag([$provider::class], SitemapRegistry::TAG);
    app()->instance($provider::class, $provider);
    app()->forgetInstance(SitemapRegistry::class);
}

it('serves a valid sitemap index of non-empty providers on the canonical https origin', function (): void {
    $category = Category::factory()->create(['slug' => 'cycle']);
    Post::factory()->published(Carbon::parse('2026-09-20 09:00:00'))->create(['slug' => 'period-pain', 'category_id' => $category->id]);

    $xml = sitemapXml($this->get('/sitemap.xml'));

    expect($xml->getName())->toBe('sitemapindex')
        ->and($xml->getNamespaces())->toBe(['' => 'http://www.sitemaps.org/schemas/sitemap/0.9'])
        ->and(sitemapLocs($xml))->toBe([
            'https://ritme.ir/sitemaps/pages.xml',
            'https://ritme.ir/sitemaps/posts.xml',
            'https://ritme.ir/sitemaps/blog-categories.xml',
        ])
        ->and((string) $xml->sitemap[1]->lastmod)->toBe('2026-09-20T09:00:00+03:30')
        ->and(isset($xml->sitemap[0]->lastmod))->toBeFalse();
});

it('lists only indexable static pages, honouring seo_meta robots and sitemap_include', function (): void {
    SeoMeta::query()->create(['route_name' => 'terms', 'robots' => 'noindex, follow']);
    SeoMeta::query()->create(['route_name' => 'privacy', 'sitemap_include' => false]);
    SeoMeta::query()->create(['route_name' => 'about', 'sitemap_priority' => 0.7, 'sitemap_changefreq' => 'weekly']);

    $xml = sitemapXml($this->get('/sitemaps/pages.xml'));
    $locs = sitemapLocs($xml);

    expect($xml->getName())->toBe('urlset')
        ->and($locs)->toContain('https://ritme.ir', 'https://ritme.ir/cycle', 'https://ritme.ir/blog', 'https://ritme.ir/directory/business')
        ->and($locs)->not->toContain(
            'https://ritme.ir/terms',
            'https://ritme.ir/privacy',
            'https://ritme.ir/shop/cart',
            'https://ritme.ir/shop/checkout',
            'https://ritme.ir/directory/join',
            'https://ritme.ir/directory/join/done',
        )
        ->and($locs)->each->toStartWith('https://ritme.ir');

    $urls = [];
    foreach ($xml->url as $url) {
        $urls[(string) $url->loc] = $url;
    }
    expect((string) $urls['https://ritme.ir/about']->priority)->toBe('0.7')
        ->and((string) $urls['https://ritme.ir/about']->changefreq)->toBe('weekly')
        ->and((string) $urls['https://ritme.ir']->priority)->toBe('1.0')
        ->and(isset($urls['https://ritme.ir']->lastmod))->toBeFalse();
});

it('lists indexable posts with real lastmod and image:image, excluding noindex posts', function (): void {
    $media = Media::query()->create([
        'disk' => 'public', 'directory' => '2026/10/abc', 'filename' => 'cover.jpg', 'mime' => 'image/jpeg',
        'size' => 100, 'width' => 1200, 'height' => 630, 'alt' => 'درد پریود', 'hash' => str_repeat('a', 64),
    ]);
    Carbon::setTestNow('2026-10-04 12:00:00');
    Post::factory()->published(now()->subDay())->create(['slug' => 'period-pain', 'cover_media_id' => $media->id]);
    $hidden = Post::factory()->published()->create(['slug' => 'hidden']);
    SeoMeta::query()->create(['seoable_type' => $hidden->getMorphClass(), 'seoable_id' => $hidden->id, 'robots' => 'noindex']);

    $xml = sitemapXml($this->get('/sitemaps/posts.xml'));
    $xml->registerXPathNamespace('image', 'http://www.google.com/schemas/sitemap-image/1.1');

    expect(sitemapLocs($xml))->toBe(['https://ritme.ir/blog/period-pain'])
        ->and((string) $xml->url[0]->lastmod)->toBe('2026-10-03T12:00:00+03:30')
        ->and(array_map('strval', $xml->xpath('//image:image/image:loc') ?: []))->toBe(['https://ritme.ir/media/2026/10/abc/cover.jpg']);
});

it('caches both documents: a warm request runs zero queries and content changes refresh them', function (): void {
    Post::factory()->published()->create(['slug' => 'first']);

    foreach (['/sitemap.xml', '/sitemaps/pages.xml', '/sitemaps/posts.xml', '/robots.txt'] as $path) {
        $this->get($path)->assertOk();
        expect(sitemapQueryCount(fn () => $this->get($path)->assertOk()))->toBe(0, $path);
    }

    Post::factory()->published()->create(['slug' => 'second']);

    expect(sitemapLocs(sitemapXml($this->get('/sitemaps/posts.xml'))))->toContain('https://ritme.ir/blog/second');
});

it('splits providers into files of 5000 URLs', function (): void {
    registerFakeSitemapProvider(new class implements SitemapProvider
    {
        public function key(): string
        {
            return 'fake-items';
        }

        public function count(): int
        {
            return Sitemaps::PER_PAGE + 1;
        }

        public function entries(int $page = 1, int $perPage = 5000): array
        {
            $from = ($page - 1) * $perPage + 1;
            $to = min($this->count(), $page * $perPage);

            return $from > $to ? [] : array_map(static fn (int $i): SitemapEntryData => new SitemapEntryData("http://localhost/items/{$i}"), range($from, $to));
        }
    });

    expect(sitemapLocs(sitemapXml($this->get('/sitemap.xml'))))->toContain('https://ritme.ir/sitemaps/fake-items.xml', 'https://ritme.ir/sitemaps/fake-items-2.xml')
        ->and(sitemapXml($this->get('/sitemaps/fake-items.xml'))->url)->toHaveCount(Sitemaps::PER_PAGE)
        ->and(sitemapLocs(sitemapXml($this->get('/sitemaps/fake-items-2.xml'))))->toBe(['https://ritme.ir/items/5001']);

    $this->get('/sitemaps/fake-items-3.xml')->assertNotFound();
    $this->get('/sitemaps/fake-items-1.xml')->assertNotFound(); // page 1 has no suffix
});

it('returns 404 for unknown or malformed sitemap files', function (string $path): void {
    $this->get($path)->assertNotFound();
})->with(['/sitemaps/unknown.xml', '/sitemaps/pages-1.xml', '/sitemaps/pages-2.xml', '/sitemaps/Pages.xml', '/sitemaps/pages-.xml']);

it('rejects providers with invalid keys', function (): void {
    registerFakeSitemapProvider(new class implements SitemapProvider
    {
        public function key(): string
        {
            return 'items-2';
        }

        public function count(): int
        {
            return 0;
        }

        public function entries(int $page = 1, int $perPage = 5000): array
        {
            return [];
        }
    });

    app(SitemapRegistry::class)->all();
})->throws(InvalidArgumentException::class, 'Invalid sitemap provider key [items-2].');

it('disallows everything outside production', function (string $env): void {
    config(['app.env' => $env]);

    $response = $this->get('/robots.txt');

    $response->assertOk()->assertHeader('Content-Type', 'text/plain; charset=UTF-8');
    expect($response->getContent())->toBe("User-agent: *\nDisallow: /\n")
        ->and($response->headers->getCookies())->toBe([]);
})->with(['local', 'staging', 'testing']);

it('serves the configured rules and the sitemap line in production', function (): void {
    config(['app.env' => 'production']);

    $body = (string) $this->get('/robots.txt')->assertOk()->getContent();

    expect($body)->toStartWith("User-agent: *\n")
        ->and($body)->toContain("Disallow: /admin\n", "Disallow: /shop/checkout\n")
        ->and($body)->not->toContain("Disallow: /\n")
        ->and($body)->toEndWith("\n\nSitemap: https://ritme.ir/sitemap.xml\n");
});

it('replaces the static public/robots.txt and sets no cookies on crawler files', function (): void {
    expect(file_exists(public_path('robots.txt')))->toBeFalse()
        ->and($this->get('/sitemap.xml')->headers->getCookies())->toBe([]);
});

it('redirects old WordPress sitemap indexes', function (string $path): void {
    $this->get($path)->assertStatus(301)->assertRedirect('/sitemap.xml');
})->with(['/sitemap_index.xml', '/wp-sitemap.xml']);
