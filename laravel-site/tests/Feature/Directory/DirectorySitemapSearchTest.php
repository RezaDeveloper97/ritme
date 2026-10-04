<?php

declare(strict_types=1);

use App\Domain\Directory\Models\City;
use App\Domain\Directory\Models\Place;
use App\Domain\Directory\Models\PlaceCategory;
use App\Domain\Directory\Search\PlaceSearchProvider;
use App\Domain\Directory\Sitemap\LandingSitemapProvider;
use App\Domain\Directory\Sitemap\PlaceSitemapProvider;
use App\Domain\Search\Actions\SearchSite;
use App\Domain\Search\Support\SearchRegistry;
use App\Domain\Seo\Models\SeoMeta;
use App\Domain\Seo\Sitemap\SitemapRegistry;

beforeEach(function (): void {
    config(['app.url' => 'https://ritme.ir']);
    url()->forceRootUrl('https://ritme.ir');
    url()->forceScheme('https');
    $this->tehran = City::factory()->create(['slug' => 'tehran', 'name' => 'تهران']);
    $this->pool = PlaceCategory::factory()->create(['slug' => 'pool', 'name' => 'استخر مادر و کودک']);
});

it('registers the directory sitemaps and search provider', function (): void {
    $keys = array_map(static fn ($p): string => $p->key(), app(SitemapRegistry::class)->all());

    expect($keys)->toContain('directory-places', 'directory-landings')
        ->and(array_map(static fn ($p): string => $p::class, app(SearchRegistry::class)->all()))->toContain(PlaceSearchProvider::class);
});

it('lists indexable, published, non-demo places in the sitemap', function (): void {
    $base = ['city_id' => $this->tehran->id, 'category_id' => $this->pool->id];
    Place::factory()->published()->create([...$base, 'slug' => 'ab-pari']);
    Place::factory()->published()->demo()->create([...$base, 'slug' => 'demo']);
    Place::factory()->create([...$base, 'slug' => 'draft']);
    $hidden = Place::factory()->published()->create([...$base, 'slug' => 'hidden']);
    SeoMeta::query()->create(['seoable_type' => $hidden->getMorphClass(), 'seoable_id' => $hidden->id, 'robots' => 'noindex, follow']);

    $provider = app(PlaceSitemapProvider::class);
    $entries = $provider->entries();

    expect($provider->count())->toBe(1)
        ->and($entries[0]->loc)->toBe('https://ritme.ir/directory/place/ab-pari')
        ->and($entries[0]->lastmod)->not->toBeNull()
        ->and($hidden->getMorphClass())->toBe('directory_place');

    Place::factory()->published()->create([...$base, 'slug' => 'new-one']);
    expect($provider->count())->toBe(2);
});

it('lists only landing pages with at least one real place', function (): void {
    $class = PlaceCategory::factory()->create(['slug' => 'class']);
    Place::factory()->published()->create(['city_id' => $this->tehran->id, 'category_id' => $this->pool->id]);
    Place::factory()->published()->demo()->create(['city_id' => $this->tehran->id, 'category_id' => $class->id]);

    $locs = array_map(static fn ($e): string => $e->loc, app(LandingSitemapProvider::class)->entries());

    expect($locs)->toBe(['https://ritme.ir/directory/tehran', 'https://ritme.ir/directory/tehran/pool']);

    $this->get('/sitemaps/directory-landings.xml')->assertOk()->assertSee('https://ritme.ir/directory/tehran/pool', false);
});

it('finds places in site search', function (): void {
    Place::factory()->published()->create([
        'city_id' => $this->tehran->id, 'category_id' => $this->pool->id, 'slug' => 'ab-pari',
        'name' => 'استخر مادر و کودک آب‌پری', 'summary' => 'استخر سرپوشیده با آب گرم.',
    ]);
    Place::factory()->create(['city_id' => $this->tehran->id, 'category_id' => $this->pool->id, 'name' => 'آب‌پری پیش‌نویس']);

    $hits = array_values(array_filter(app(SearchSite::class)->handle('آب پری')->hits, static fn ($h): bool => $h->type === 'places'));

    expect($hits)->toHaveCount(1)
        ->and($hits[0]->title)->toBe('استخر مادر و کودک آب‌پری')
        ->and($hits[0]->url)->toBe('https://ritme.ir/directory/place/ab-pari')
        ->and($hits[0]->typeLabel)->toBe('خدمات مادر و کودک')
        ->and($hits[0]->snippet)->toContain('تهران');
});
