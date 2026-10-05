<?php

declare(strict_types=1);

use Tests\Arch\SourceFiles;

/*
 * Dependency rule: Http/Filament -> Actions/Contracts -> Models. See docs/ARCHITECTURE.md.
 *
 * Token scans (Tests\Arch\SourceFiles) instead of Pest's arch() expectations: those parse and keep every AST of
 * app/ (~550 MB RSS, ~6 s); this keeps the whole suite inside phpunit.xml's 512M.
 */

/**
 * @param  list<string>  $forbidden
 * @return list<string>
 */
function layerViolations(string $directory, array $forbidden): array
{
    $violations = [];
    foreach (SourceFiles::in($directory) as $path) {
        foreach (SourceFiles::forbiddenNames($path, $forbidden) as $name) {
            $violations[] = SourceFiles::relative($path).' uses '.$name;
        }
    }

    return $violations;
}

it('keeps the domain free of delivery layers', function (): void {
    expect(layerViolations('app/Domain', ['Illuminate\Http', 'App\Http', 'App\Filament']))->toBe([]);
});

it('keeps the support kernel free of domain and delivery layers', function (): void {
    expect(layerViolations('app/Support', ['App\Domain', 'App\Http', 'App\Filament']))->toBe([]);
});

it('keeps controllers final', function (): void {
    $violations = [];
    foreach (SourceFiles::in('app/Http/Controllers') as $path) {
        foreach (SourceFiles::classes($path) as [$name, $final, $abstract]) {
            if (! $final && ! $abstract && $name !== 'Controller') {
                $violations[] = SourceFiles::relative($path).': '.$name;
            }
        }
    }

    expect($violations)->toBe([]);
});

it('keeps database access out of controllers', function (): void {
    $forbidden = [
        'DB',
        'Illuminate\Support\Facades\DB',
        'Illuminate\Database\DatabaseManager',
        'Illuminate\Database\Query\Builder',
        'Illuminate\Database\Eloquent\Builder',
    ];
    $violations = layerViolations('app/Http/Controllers', $forbidden);
    // A bare `DB::` (root alias) is a plain T_STRING, not a qualified name.
    foreach (SourceFiles::in('app/Http/Controllers') as $path) {
        if (preg_match('/(?<![\\\\\w])DB::/', (string) file_get_contents($path)) === 1) {
            $violations[] = SourceFiles::relative($path).' uses DB::';
        }
    }

    expect($violations)->toBe([]);
});

it('leaves no debugging calls behind', function (): void {
    $violations = [];
    foreach (['app', 'config', 'database', 'routes'] as $directory) {
        foreach (SourceFiles::in($directory) as $path) {
            foreach (SourceFiles::functionCalls($path, ['dd', 'ddd', 'dump', 'ray', 'var_dump', 'dump_server']) as $line) {
                $violations[] = SourceFiles::relative($path).':'.$line;
            }
        }
    }

    expect($violations)->toBe([]);
});

it('declares strict types everywhere', function (): void {
    $violations = [];
    foreach (['app', 'database'] as $directory) {
        foreach (SourceFiles::in($directory) as $path) {
            if (! str_ends_with($path, '.blade.php') && ! SourceFiles::declaresStrictTypes($path)) {
                $violations[] = SourceFiles::relative($path);
            }
        }
    }

    expect($violations)->toBe([]);
});

it('catches violations (scanner self-test)', function (): void {
    $file = tempnam(sys_get_temp_dir(), 'arch').'.php';
    file_put_contents($file, <<<'PHP'
        <?php
        namespace App\Domain\Demo;
        use Illuminate\Http\Request;
        class Leaky { public function x() { dump(1); $this->dump(); return \App\Filament\Thing::class; } }
        PHP);

    try {
        expect(SourceFiles::forbiddenNames($file, ['Illuminate\Http', 'App\Filament']))
            ->toBe(['Illuminate\Http\Request', 'App\Filament\Thing'])
            ->and(SourceFiles::functionCalls($file, ['dump']))->toBe([4])
            ->and(SourceFiles::declaresStrictTypes($file))->toBeFalse()
            ->and(SourceFiles::classes($file))->toBe([['Leaky', false, false]]);
    } finally {
        @unlink($file);
    }
});
