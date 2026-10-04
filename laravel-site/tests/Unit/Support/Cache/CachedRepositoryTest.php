<?php

declare(strict_types=1);

use App\Support\Cache\CacheAside;
use App\Support\Cache\CachedRepository;
use App\Support\Cache\NamespaceVersions;
use Tests\TestCase;
use Tests\Unit\Support\Cache\CacheStores;

uses(TestCase::class);

final class CacheAsideFixtureSource
{
    public int $calls = 0;

    public function find(string $slug): ?string
    {
        $this->calls++;

        return $slug === 'missing' ? null : "post:{$slug}";
    }
}

final class CacheAsideFixtureCachedRepository extends CachedRepository
{
    public function __construct(public readonly CacheAsideFixtureSource $inner, CacheAside $cache)
    {
        parent::__construct($cache);
    }

    protected function namespace(): string
    {
        return 'blog';
    }

    public function find(string $slug): ?string
    {
        return $this->remember(['post', $slug], fn (): ?string => $this->inner->find($slug));
    }

    public function findOrNull(string $slug): ?string
    {
        return $this->remember("null:{$slug}", fn (): ?string => $this->inner->find($slug), 60, cacheNull: true);
    }

    public function findForever(int $id): ?string
    {
        return $this->rememberForever($id, fn (): ?string => $this->inner->find((string) $id));
    }

    public function evict(string $slug): bool
    {
        return $this->forget(['post', $slug]);
    }
}

beforeEach(function (): void {
    config(['cacheaside.namespaces' => ['blog' => 3600]]);
    CacheStores::use('array');
    $this->repo = app(CacheAsideFixtureCachedRepository::class, ['inner' => new CacheAsideFixtureSource]);
});

it('caches reads of the inner repository under its namespace', function (): void {
    expect($this->repo->find('a'))->toBe('post:a')
        ->and($this->repo->find('a'))->toBe('post:a')
        ->and($this->repo->inner->calls)->toBe(1)
        ->and(app(NamespaceVersions::class)->store()->get('rt:blog:v1:post:a'))->toBe('post:a');
});

it('supports forever, null caching and forget', function (): void {
    $this->repo->findForever(5);
    $this->repo->findForever(5);
    $this->repo->findOrNull('missing');
    $this->repo->findOrNull('missing');
    expect($this->repo->inner->calls)->toBe(2);

    $this->repo->find('b');
    expect($this->repo->evict('b'))->toBeTrue();
    $this->repo->find('b');
    expect($this->repo->inner->calls)->toBe(4);
});

it('is invalidated by a namespace bump', function (): void {
    $this->repo->find('a');
    app(NamespaceVersions::class)->bump('blog');
    $this->repo->find('a');

    expect($this->repo->inner->calls)->toBe(2);
});
