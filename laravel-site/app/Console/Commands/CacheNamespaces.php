<?php

declare(strict_types=1);

namespace App\Console\Commands;

use App\Support\Cache\NamespaceVersions;
use Illuminate\Console\Command;
use InvalidArgumentException;

final class CacheNamespaces extends Command
{
    protected $signature = 'cache:ns
        {action=list : list | bump | bump-all}
        {namespace?* : namespace(s) to bump}';

    protected $description = 'List or bump the versioned cache-aside namespaces';

    public function handle(NamespaceVersions $versions): int
    {
        $action = (string) $this->argument('action');

        return match ($action) {
            'list' => $this->list($versions),
            'bump' => $this->bump($versions),
            'bump-all' => $this->report($versions->bumpAll()),
            default => $this->fail("Unknown action [{$action}]. Use list, bump or bump-all."),
        };
    }

    private function list(NamespaceVersions $versions): int
    {
        $rows = [];
        foreach ($versions->all() as $namespace) {
            $rows[] = [$namespace->name, 'v'.$namespace->version, $namespace->ttl.'s'];
        }

        $this->table(['Namespace', 'Version', 'Default TTL'], $rows);

        return self::SUCCESS;
    }

    private function bump(NamespaceVersions $versions): int
    {
        /** @var list<string> $namespaces */
        $namespaces = (array) $this->argument('namespace');

        if ($namespaces === []) {
            $this->error('Give at least one namespace, e.g. php artisan cache:ns bump blog');

            return self::FAILURE;
        }

        try {
            return $this->report($versions->bump(...$namespaces));
        } catch (InvalidArgumentException $e) {
            $this->error($e->getMessage());

            return self::FAILURE;
        }
    }

    /**
     * @param  array<string, int>  $bumped
     */
    private function report(array $bumped): int
    {
        foreach ($bumped as $namespace => $version) {
            $this->info("{$namespace} → v{$version}");
        }

        return self::SUCCESS;
    }
}
