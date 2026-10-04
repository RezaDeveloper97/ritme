<?php

declare(strict_types=1);

namespace App\Http\Middleware;

use Illuminate\Contracts\Config\Repository as Config;
use Illuminate\Http\Request;

/**
 * Request paths that belong to the Filament admin: filament.admin.path plus pagecache.security.admin_paths
 * (Livewire and Filament endpoints). They get the relaxed CSP, are never minified and never page-cached.
 */
final class AdminPaths
{
    public static function matches(Request $request, Config $config): bool
    {
        return $request->is(...self::patterns($config));
    }

    /**
     * @return list<string>
     */
    public static function patterns(Config $config): array
    {
        $admin = trim((string) $config->get('filament.admin.path', 'admin'), '/');
        $paths = array_values(array_map('strval', (array) $config->get('pagecache.security.admin_paths', [])));

        return $admin === '' ? $paths : [...$paths, $admin, $admin.'/*'];
    }
}
