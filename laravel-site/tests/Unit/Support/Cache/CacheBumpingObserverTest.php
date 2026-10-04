<?php

declare(strict_types=1);

use App\Support\Cache\CacheBumpingObserver;
use App\Support\Cache\NamespaceBumper;
use App\Support\Cache\NamespaceVersions;
use Illuminate\Database\Connection;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\SoftDeletes;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Database\SQLiteConnection;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;
use Tests\TestCase;
use Tests\Unit\Support\Cache\CacheStores;

uses(TestCase::class);

final class CacheAsideFixturePost extends Model
{
    use SoftDeletes;

    protected $table = 'cache_aside_posts';

    protected $guarded = [];
}

final class CacheAsideFixturePostObserver extends CacheBumpingObserver
{
    protected function namespaces(Model $model): array
    {
        return ['blog', 'sitemap'];
    }
}

/** @return array<string, int> */
function cacheAsideVersions(): array
{
    $versions = app(NamespaceVersions::class);
    $all = [];
    foreach (['blog', 'sitemap', 'faq', 'pages'] as $namespace) {
        $all[$namespace] = $versions->version($namespace);
    }

    return $all;
}

beforeEach(function (): void {
    config([
        'cacheaside.namespaces' => ['blog' => 60, 'sitemap' => 60, 'faq' => 60, 'pages' => 60],
        'cacheaside.always_bump' => ['pages'],
    ]);
    CacheStores::use('array');
    Schema::create('cache_aside_posts', function (Blueprint $table): void {
        $table->id();
        $table->string('title');
        $table->timestamps();
        $table->softDeletes();
    });
    CacheAsideFixturePost::observe(CacheAsideFixturePostObserver::class);
});

afterEach(function (): void {
    CacheAsideFixturePost::flushEventListeners();
});

it('bumps its namespaces and pages on saved, deleted and restored', function (): void {
    $post = CacheAsideFixturePost::query()->create(['title' => 'a']);
    expect(cacheAsideVersions())->toBe(['blog' => 2, 'sitemap' => 2, 'faq' => 1, 'pages' => 2]);

    $post->update(['title' => 'b']);
    expect(cacheAsideVersions()['blog'])->toBe(3);

    $post->delete();
    expect(cacheAsideVersions()['blog'])->toBe(4);

    $post->restore(); // restore() saves (+1) and fires restored (+1)
    expect(cacheAsideVersions())->toBe(['blog' => 6, 'sitemap' => 6, 'faq' => 1, 'pages' => 6]);
});

it('defers the bump until the transaction commits and skips it on rollback', function (): void {
    DB::transaction(function (): void {
        CacheAsideFixturePost::query()->create(['title' => 'a']);
        expect(cacheAsideVersions()['blog'])->toBe(1);
    });
    expect(cacheAsideVersions()['blog'])->toBe(2);

    try {
        DB::transaction(function (): void {
            CacheAsideFixturePost::query()->create(['title' => 'b']);
            throw new RuntimeException('rollback');
        });
    } catch (RuntimeException) {
        // expected
    }
    expect(cacheAsideVersions()['blog'])->toBe(2);
});

it('bumps only always_bump namespaces when the model has none', function (): void {
    app(NamespaceBumper::class)->bumpFor(new CacheAsideFixturePost, []);

    expect(cacheAsideVersions())->toBe(['blog' => 1, 'sitemap' => 1, 'faq' => 1, 'pages' => 2]);
});

it('bumps immediately on a connection without a transactions manager', function (): void {
    $model = new class extends Model
    {
        public function getConnection(): Connection
        {
            return new SQLiteConnection(new PDO('sqlite::memory:'));
        }
    };

    app(NamespaceBumper::class)->bumpFor($model, ['faq']);

    expect(cacheAsideVersions())->toBe(['blog' => 1, 'sitemap' => 1, 'faq' => 2, 'pages' => 2]);
});

it('does nothing when there is nothing to bump', function (): void {
    config(['cacheaside.always_bump' => []]);

    app(NamespaceBumper::class)->bumpFor(new CacheAsideFixturePost, []);

    expect(cacheAsideVersions())->toBe(['blog' => 1, 'sitemap' => 1, 'faq' => 1, 'pages' => 1]);
});
