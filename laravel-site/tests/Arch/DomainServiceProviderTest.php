<?php

declare(strict_types=1);

use App\Providers\DomainServiceProvider;
use Illuminate\Database\Eloquent\Model;
use Tests\TestCase;

uses(TestCase::class);

interface ArchFixtureRepository
{
    public function name(): string;
}

final class ArchFixtureEloquentRepository implements ArchFixtureRepository
{
    public function name(): string
    {
        return 'eloquent';
    }
}

final class ArchFixtureCachedRepository implements ArchFixtureRepository
{
    public function __construct(public readonly ArchFixtureRepository $inner) {}

    public function name(): string
    {
        return 'cached('.$this->inner->name().')';
    }
}

it('binds a contract to its cached decorator wrapping the eloquent implementation', function (): void {
    $provider = new class(app()) extends DomainServiceProvider
    {
        protected array $repositories = [
            ArchFixtureRepository::class => [ArchFixtureEloquentRepository::class, ArchFixtureCachedRepository::class],
        ];
    };
    $provider->register();

    $repository = app(ArchFixtureRepository::class);

    expect($repository)->toBeInstanceOf(ArchFixtureCachedRepository::class)
        ->and($repository->name())->toBe('cached(eloquent)')
        ->and(app(ArchFixtureRepository::class))->toBe($repository);
});

it('binds a contract straight to the eloquent implementation when no decorator is given', function (): void {
    $provider = new class(app()) extends DomainServiceProvider
    {
        protected array $repositories = [
            ArchFixtureRepository::class => ArchFixtureEloquentRepository::class,
        ];
    };
    $provider->register();

    expect(app(ArchFixtureRepository::class)->name())->toBe('eloquent');
});

it('registers every bounded-context provider', function (): void {
    foreach (['Seo', 'Settings', 'Media', 'Content', 'Faq', 'Contact', 'Blog', 'Newsletter', 'Directory', 'Shop', 'Pwa'] as $context) {
        expect(app()->getProvider("App\\Providers\\Domain\\{$context}ServiceProvider"))->not->toBeNull();
    }
});

it('enforces strict eloquent models outside production', function (): void {
    expect(Model::preventsLazyLoading())->toBeTrue()
        ->and(Model::preventsSilentlyDiscardingAttributes())->toBeTrue();
});
