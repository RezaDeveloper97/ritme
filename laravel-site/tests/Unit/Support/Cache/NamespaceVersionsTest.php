<?php

declare(strict_types=1);

use App\Support\Cache\CacheKey;
use App\Support\Cache\CacheNamespace;
use App\Support\Cache\NamespaceVersions;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Tests\TestCase;
use Tests\Unit\Support\Cache\CacheStores;

uses(TestCase::class, RefreshDatabase::class);

beforeEach(function (): void {
    config(['cacheaside.namespaces' => ['blog' => 3600, 'faq' => 60, 'pages' => 600]]);
});

it('starts every namespace at version 1', function (string $driver): void {
    CacheStores::use($driver);
    $versions = app(NamespaceVersions::class);

    expect($versions->version('blog'))->toBe(1)
        ->and($versions->qualify(CacheKey::make('blog', 'post', 7)))->toBe('rt:blog:v1:post:7');
})->with(CacheStores::drivers());

it('bumps versions independently per namespace', function (string $driver): void {
    CacheStores::use($driver);
    $versions = app(NamespaceVersions::class);

    expect($versions->bump('blog'))->toBe(['blog' => 2])
        ->and($versions->bump('blog', 'blog', 'faq'))->toBe(['blog' => 3, 'faq' => 2])
        ->and($versions->version('blog'))->toBe(3)
        ->and($versions->version('pages'))->toBe(1)
        ->and(app(NamespaceVersions::class)->qualify(CacheKey::make('blog', 'x')))->toBe('rt:blog:v3:x');
})->with(CacheStores::drivers());

it('bumps every namespace at once', function (string $driver): void {
    CacheStores::use($driver);
    $versions = app(NamespaceVersions::class);
    $versions->bump('faq');

    expect($versions->bumpAll())->toBe(['blog' => 2, 'faq' => 3, 'pages' => 2]);
})->with(CacheStores::drivers());

it('lists declared namespaces with ttl and version', function (): void {
    CacheStores::use('array');
    $versions = app(NamespaceVersions::class);
    $versions->bump('faq');

    expect($versions->all())->toEqual([
        new CacheNamespace('blog', 3600, 1),
        new CacheNamespace('faq', 60, 2),
        new CacheNamespace('pages', 600, 1),
    ])
        ->and($versions->has('blog'))->toBeTrue()
        ->and($versions->has('nope'))->toBeFalse()
        ->and($versions->ttl('faq'))->toBe(60);
});

it('rejects undeclared namespaces', function (Closure $call): void {
    CacheStores::use('array');
    $call(app(NamespaceVersions::class));
})->throws(InvalidArgumentException::class, 'not declared')->with([
    'version' => [fn (NamespaceVersions $v) => $v->version('nope')],
    'bump' => [fn (NamespaceVersions $v) => $v->bump('nope')],
    'ttl' => [fn (NamespaceVersions $v) => $v->ttl('nope')],
    'qualify' => [fn (NamespaceVersions $v) => $v->qualify(CacheKey::make('nope', 'x'))],
]);

it('honours a custom prefix and the default store', function (): void {
    config(['cacheaside.prefix' => 'site', 'cacheaside.store' => null]);
    $versions = app(NamespaceVersions::class);

    expect($versions->prefix())->toBe('site')
        ->and($versions->store()->getStore())->toBe(cache()->store()->getStore())
        ->and($versions->qualify(CacheKey::make('blog', 'x')))->toBe('site:blog:v1:x');
});

it('ignores a corrupted version value', function (): void {
    CacheStores::use('array');
    $versions = app(NamespaceVersions::class);
    $versions->store()->forever('rt:nsv:blog', 'garbage');

    expect($versions->version('blog'))->toBe(1);
});
