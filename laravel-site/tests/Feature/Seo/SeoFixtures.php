<?php

declare(strict_types=1);

namespace Tests\Feature\Seo;

use App\Domain\Seo\Concerns\HasSeo;
use App\Domain\Seo\Models\SeoMeta;
use Database\Seeders\SettingsSeeder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Facades\View;

/**
 * A model with a public page, for SeoManager::for().
 */
final class SeoFixturePage extends Model
{
    use HasSeo;

    protected $table = 'seo_fixture_pages';
}

final class SeoFixtures
{
    public static function boot(): void
    {
        test()->seed(SettingsSeeder::class);
        config(['app.url' => 'https://ritme.test']);
        View::addNamespace('seo-fixtures', __DIR__.'/fixtures');
    }

    /**
     * @param  array<string, mixed>  $attributes
     */
    public static function routeMeta(string $routeName, array $attributes): SeoMeta
    {
        return SeoMeta::query()->create(['route_name' => $routeName, ...$attributes]);
    }

    public static function page(int $id): SeoFixturePage
    {
        $page = new SeoFixturePage;
        $page->forceFill(['id' => $id]);
        $page->exists = true;

        return $page;
    }
}
