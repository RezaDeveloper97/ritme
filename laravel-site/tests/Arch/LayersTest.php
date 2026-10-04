<?php

declare(strict_types=1);

/*
 * Dependency rule: Http/Filament -> Actions/Contracts -> Models. See docs/ARCHITECTURE.md.
 */

arch('domain does not depend on delivery layers')
    ->expect('App\Domain')
    ->not->toUse(['Illuminate\Http', 'App\Http', 'App\Filament']);

arch('support kernel does not depend on domain or delivery layers')
    ->expect('App\Support')
    ->not->toUse(['App\Domain', 'App\Http', 'App\Filament']);

arch('controllers are final')
    ->expect('App\Http\Controllers')
    ->classes()
    ->toBeFinal()
    ->ignoring('App\Http\Controllers\Controller');

arch('controllers do not query the database directly')
    ->expect('App\Http\Controllers')
    ->not->toUse([
        'DB',
        'Illuminate\Support\Facades\DB',
        'Illuminate\Database\DatabaseManager',
        'Illuminate\Database\Query\Builder',
        'Illuminate\Database\Eloquent\Builder',
    ]);

arch('no debugging leftovers')
    ->expect(['dd', 'ddd', 'dump', 'ray', 'var_dump', 'dump_server'])
    ->not->toBeUsed();

arch('strict types everywhere')
    ->expect(['App', 'Database'])
    ->toUseStrictTypes();
