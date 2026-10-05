<?php

declare(strict_types=1);

use Tests\Arch\SourceFiles;

$root = SourceFiles::root();

it('calls env() only inside config/', function () use ($root): void {
    $violations = [];
    foreach (['app', 'bootstrap', 'database', 'routes', 'resources/views'] as $dir) {
        foreach (SourceFiles::in($dir) as $path) {
            if (str_contains($path, '/bootstrap/cache/')) {
                continue;
            }
            $tokens = SourceFiles::tokens($path);
            foreach ($tokens as $i => $token) {
                if (! is_array($token) || $token[0] !== T_STRING || strtolower($token[1]) !== 'env') {
                    continue;
                }
                $prev = $tokens[$i - 1] ?? null;
                if (is_array($prev) && in_array($prev[0], [T_OBJECT_OPERATOR, T_DOUBLE_COLON, T_FUNCTION, T_NULLSAFE_OBJECT_OPERATOR], true)) {
                    continue;
                }
                $next = $tokens[$i + 1] ?? null;
                if ($next === '(') {
                    $violations[] = str_replace("{$root}/", '', $path).':'.$token[2];
                }
            }
        }
    }

    expect($violations)->toBe([]);
});

it('keeps actions final and single-purpose (__invoke or handle)', function () use ($root): void {
    $violations = [];
    foreach (glob("{$root}/app/Domain/*/Actions", GLOB_ONLYDIR) ?: [] as $dir) {
        foreach (SourceFiles::in(substr($dir, strlen($root) + 1)) as $path) {
            $class = 'App\\'.str_replace(['/', '.php'], ['\\', ''], substr($path, strlen("{$root}/app/")));
            if (! class_exists($class)) {
                continue;
            }
            $reflection = new ReflectionClass($class);
            if ($reflection->isInterface() || $reflection->isAbstract() || $reflection->isEnum()) {
                continue;
            }
            if (! $reflection->isFinal()) {
                $violations[] = "{$class} is not final";
            }
            if (! $reflection->hasMethod('__invoke') && ! $reflection->hasMethod('handle')) {
                $violations[] = "{$class} has neither __invoke() nor handle()";
            }
        }
    }

    expect($violations)->toBe([]);
});
