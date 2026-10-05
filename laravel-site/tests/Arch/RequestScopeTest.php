<?php

declare(strict_types=1);

use App\Domain\Seo\Schema\SchemaGraph;
use App\Domain\Seo\SeoManager;
use App\Domain\Shop\Cart\Actions\AddToCart;
use Illuminate\Container\Container;
use Tests\Arch\SourceFiles;
use Tests\TestCase;

uses(TestCase::class);

/*
 * The router caches controller instances on the Route, so a controller outlives its request (in tests, Octane, the
 * in-process SEO audit crawler). Request-scoped services (SeoManager, SchemaGraph, the session cart, CheckoutSession…)
 * must therefore be method-injected, never reach a controller's constructor — directly or through a dependency.
 */

/**
 * @param  list<string>  $scoped
 * @param  array<class-string, true>  $seen
 * @return list<string> dependency paths that end in a scoped service
 */
function scopedConstructorPaths(string $class, array $scoped, array $seen = [], int $depth = 0): array
{
    if ($depth > 4 || isset($seen[$class]) || ! class_exists($class)) {
        return [];
    }
    $seen[$class] = true;
    $constructor = (new ReflectionClass($class))->getConstructor();
    $paths = [];
    foreach ($constructor?->getParameters() ?? [] as $parameter) {
        $type = $parameter->getType();
        if (! $type instanceof ReflectionNamedType || $type->isBuiltin()) {
            continue;
        }
        $dependency = $type->getName();
        $concrete = app()->bound($dependency) ? app()->getBindings()[$dependency]['concrete'] ?? null : null;
        if (in_array($dependency, $scoped, true)) {
            $paths[] = $dependency;

            continue;
        }
        $next = is_string($concrete) ? $concrete : $dependency;
        foreach (scopedConstructorPaths($next, $scoped, $seen, $depth + 1) as $path) {
            $paths[] = $dependency.' → '.$path;
        }
    }

    return $paths;
}

it('never injects request-scoped services into controller constructors', function (): void {
    $scoped = (fn (): array => $this->scopedInstances)->call(Container::getInstance());
    expect($scoped)->toContain(SeoManager::class, SchemaGraph::class);
    // Self-check: a cart action reaches the scoped session cart through its constructor.
    expect(scopedConstructorPaths(AddToCart::class, $scoped))->not->toBe([]);

    $violations = [];
    foreach (SourceFiles::in('app/Http/Controllers') as $path) {
        $class = 'App\\'.str_replace(['/', '.php'], ['\\', ''], substr(SourceFiles::relative($path), strlen('app/')));
        foreach (scopedConstructorPaths($class, $scoped) as $chain) {
            $violations[] = "{$class}: {$chain}";
        }
    }

    expect($violations)->toBe([]);
});
