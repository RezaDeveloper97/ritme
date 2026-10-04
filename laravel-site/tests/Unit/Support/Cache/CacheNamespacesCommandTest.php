<?php

declare(strict_types=1);

use App\Support\Cache\NamespaceVersions;
use Tests\TestCase;
use Tests\Unit\Support\Cache\CacheStores;

uses(TestCase::class);

beforeEach(function (): void {
    config(['cacheaside.namespaces' => ['blog' => 3600, 'faq' => 60]]);
    CacheStores::use('array');
});

it('lists namespaces with version and ttl', function (): void {
    app(NamespaceVersions::class)->bump('faq');

    $this->artisan('cache:ns')
        ->expectsTable(['Namespace', 'Version', 'Default TTL'], [['blog', 'v1', '3600s'], ['faq', 'v2', '60s']])
        ->assertSuccessful();
});

it('bumps the given namespaces', function (): void {
    $this->artisan('cache:ns bump blog faq')
        ->expectsOutput('blog → v2')
        ->expectsOutput('faq → v2')
        ->assertSuccessful();

    expect(app(NamespaceVersions::class)->version('blog'))->toBe(2);
});

it('bumps every namespace', function (): void {
    $this->artisan('cache:ns bump-all')
        ->expectsOutput('blog → v2')
        ->expectsOutput('faq → v2')
        ->assertSuccessful();
});

it('fails on missing, unknown or undeclared input', function (string $command): void {
    $this->artisan($command)->assertFailed();
})->with([
    'bump without namespace' => ['cache:ns bump'],
    'undeclared namespace' => ['cache:ns bump nope'],
    'unknown action' => ['cache:ns frobnicate'],
]);
